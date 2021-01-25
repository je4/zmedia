package mediaserver

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/goph/emperror"
	"github.com/gorilla/mux"
	"github.com/je4/zmedia/v2/pkg/database"
	"github.com/je4/zmedia/v2/pkg/filesystem"
	"github.com/je4/zmedia/v2/pkg/media"
	"github.com/op/go-logging"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type MediaHandler struct {
	log        *logging.Logger
	mdb        *database.MediaDatabase
	fss        map[string]filesystem.FileSystem
	action     map[string]media.Action
	prefix     string
	idx        *Indexer
	pbx        ParamBuilder
	tempfolder string
}

func _buildFilename(coll *database.Collection, master *database.Master, action string, params []string) string {
	return fmt.Sprintf("%v.%v-%x", coll.Id, master.Id, md5.Sum([]byte(fmt.Sprintf("%s/%s/%s/%s", coll.Name, master.Signature, action, strings.Join(params, "/")))))
}

func buildFilename(coll *database.Collection, master *database.Master, action string, paramstr string) string {
	return fmt.Sprintf("%v.%v-%x", coll.Id, master.Id, md5.Sum([]byte(fmt.Sprintf("%s/%s/%s/%s", coll.Name, master.Signature, action, paramstr))))
}

func NewMediaHandler(
	prefix string,
	mdb *database.MediaDatabase,
	idx *Indexer,
	pbx ParamBuilder,
	tempdir string,
	log *logging.Logger,
	fss []filesystem.FileSystem,
	actions []media.Action) (*MediaHandler, error) {
	mh := &MediaHandler{
		log:    log,
		prefix: prefix,
		mdb:    mdb,
		fss:    make(map[string]filesystem.FileSystem),
		pbx:    pbx,
		idx:    idx,
		action: make(map[string]media.Action),
	}
	mh.idx.SetMediaHandler(mh)
	for _, fs := range fss {
		mh.fss[fs.Protocol()] = fs
	}
	for _, action := range actions {
		mh.action[action.GetType()] = action
	}

	fs, bucket, path, err := mh.GetFS(tempdir)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot interpret tempdir %s", tempdir)
	}
	if !fs.IsLocal() {
		return nil, fmt.Errorf("temp folder %s not local", tempdir)
	}
	url, err := fs.GETUrl(bucket, path, 0)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot get url of tempdir %s", tempdir)
	}
	mh.tempfolder = url.Path

	return mh, nil
}

var pathRegexp = regexp.MustCompile(`^([^:]+://[^/]+)/([^/]+)(/.+)?$`)

func (mh *MediaHandler) GetFS(path string) (filesystem.FileSystem, string, string, error) {
	matches := pathRegexp.FindStringSubmatch(path)
	if matches == nil {
		return nil, "", "", fmt.Errorf("invalid path - cannot load file %s from storage", path)
	}
	fs, ok := mh.fss[matches[1]]
	if !ok {
		return nil, "", "", fmt.Errorf("invalid protocol - cannot find storage %s", matches[1])
	}
	return fs, matches[2], strings.TrimLeft(matches[3], "/"), nil
}

func (mh *MediaHandler) FileOpenRead(path string, opts filesystem.FileGetOptions) (filesystem.ReadSeekerCloser, os.FileInfo, error) {
	matches := pathRegexp.FindStringSubmatch(path)
	if matches == nil {
		return nil, nil, fmt.Errorf("invalid path - cannot load file %s from storage", path)
	}
	fs, ok := mh.fss[matches[1]]
	if !ok {
		return nil, nil, fmt.Errorf("invalid protocol - cannot find storage %s", matches[1])
	}
	return fs.FileOpenRead(matches[2], matches[3], opts)
}
func (mh *MediaHandler) FileWrite(path string, reader io.Reader, size int64, opts filesystem.FilePutOptions) error {
	fs, bucket, path, err := mh.GetFS(path)
	if err != nil {
		return err
	}
	return fs.FileWrite(bucket, path, reader, size, opts)
}

var errorTemplate = template.Must(template.New("error").Parse(`<html>
<head><title>{{.Error}}</title></head>
<body><h1>{{.Error}}</h1><h2>{{.Message}}</h2></body>
</html>
`))

func (mh *MediaHandler) DoPanicf(writer http.ResponseWriter, status int, message string, jsonresult bool, a ...interface{}) {
	msg := fmt.Sprintf(message, a...)
	errorstatus := struct {
		Error   string
		Message string
	}{
		Error:   fmt.Sprintf("%v - %s", status, http.StatusText(status)),
		Message: msg,
	}
	mh.log.Errorf("error: %s // %s", errorstatus.Message, errorstatus.Error)
	writer.WriteHeader(status)
	if jsonresult {
		enc := json.NewEncoder(writer)
		enc.Encode(errorstatus)
		return
	} else {
		errorTemplate.Execute(writer, errorstatus)
	}

	return
}

func (s *MediaHandler) DoPanic(writer http.ResponseWriter, status int, message string, jsonresult bool) {
	s.DoPanicf(writer, status, message, jsonresult)
}

func (mh *MediaHandler) ServeContent(w http.ResponseWriter, r *http.Request, path string) {
	reader, finfo, err := mh.FileOpenRead(path, filesystem.FileGetOptions{})
	if err != nil {
		mh.DoPanicf(w, http.StatusNotFound, "cannot open %s: %v", false, path, err)
		return
	}
	defer reader.Close()
	http.ServeContent(w, r, finfo.Name(), finfo.ModTime(), reader)
}

func (mh *MediaHandler) GetCache(collection, signature, action, paramstr string) (*database.Cache, error) {
	// clear parameters
	params, err := mh.pbx.Clear(action, strings.Split(paramstr, "/"))
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot clear params %s for action %s", paramstr, action)
	}
	// rebuild paramstring (sorted)
	ps := []string{}
	for key, val := range params {
		ps = append(ps, key+val)
	}
	sort.Strings(ps)

	paramstr = strings.Join(ps, "/")
	cache, err := mh.mdb.GetCache(collection, signature, action, paramstr)
	if err == database.ErrNotFound {
		if action == "master" {
			master, cache, err := mh.ingestMaster(collection, signature)
			if err != nil {
				return nil, emperror.Wrapf(err, "cannot ingest master %s/%s", collection, signature)
			}
			mh.log.Infof("master: %v // %v", master, cache)
			cache, err = mh.mdb.GetCache(collection, signature, action, paramstr)
		} else {
			coll, err := mh.mdb.GetCollectionByName(collection)
			if err != nil {
				return nil, emperror.Wrapf(err, "invalid collection %s", collection)
			}
			stor, err := coll.GetStorage()
			if err != nil {
				return nil, emperror.Wrapf(err, "cannot get storage #%v from collection %s", coll.StorageId, collection)
			}
			master, err := mh.mdb.GetMaster(coll, signature)
			if err != nil {
				return nil, emperror.Wrapf(err, "cannot load master %s/%s", collection, signature)
			}
			act, ok := mh.action[master.Type]
			if !ok {
				return nil, fmt.Errorf("invalid type %s for %s/%s", master.Type, collection, signature)
			}
			mastercache, err := mh.mdb.GetCacheByMaster(master, "master", "")
			if err != nil {
				// ingest???
				return nil, emperror.Wrapf(err, "cannot load master cache of %s/%s", collection, signature)
			}
			file, _, err := mh.FileOpenRead(mastercache.Path, filesystem.FileGetOptions{})
			if err != nil {
				return nil, emperror.Wrapf(err, "open master cache file %s of %s/%s", mastercache.Path, collection, signature)
			}
			filename := buildFilename(coll, master, action, paramstr)
			bucket, err := stor.GetBucket()
			if err != nil {
				return nil, emperror.Wrapf(err, "cannot get bucket from stor %s - %s", stor.Name, stor.Filebase)
			}
			cm, err := act.Do(master, action, params, bucket, filename, file)
			cache, err = database.NewCache(
				mh.mdb,
				0,
				coll.Id,
				master.Id,
				action,
				paramstr,
				cm.Mimetype,
				cm.Size,
				fmt.Sprintf("%s/%s", "data", filename),
				cm.Width,
				cm.Height,
				0)
			if err != nil {
				return nil, emperror.Wrapf(err, "cannot create cache %s/%s/%s/%s", coll.Name, master.Signature, action, paramstr)
			}
			if err := cache.Store(); err != nil {
				return nil, emperror.Wrapf(err, "cannot store cache %s/%s/%s/%s", coll.Name, master.Signature, action, paramstr)
			}
		}
	}
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot get cache for %s/%s/%s/%s", collection, signature, action, paramstr)
	}
	return cache, err
}

func (mh *MediaHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	collection, ok := vars["collection"]
	if !ok {
		mh.DoPanicf(resp, http.StatusBadRequest, "no collection in request %s", false, req.URL.String())
		return
	}
	signature, ok := vars["signature"]
	if !ok {
		mh.DoPanicf(resp, http.StatusBadRequest, "no signature in request %s", false, req.URL.String())
		return
	}
	action, ok := vars["action"]
	if !ok {
		mh.DoPanicf(resp, http.StatusBadRequest, "no action in request %s", false, req.URL.String())
		return
	}
	paramstr, _ := vars["paramstr"]
	params := strings.Split(strings.ToLower(paramstr), "/")
	sort.Strings(params)

	cache, err := mh.GetCache(collection, signature, action, paramstr)
	switch err {
	case nil:
		resp.Header().Set("Content-type", cache.Mimetype)
		mh.ServeContent(resp, req, cache.Path)
		return
	default:
		mh.DoPanicf(resp, http.StatusBadRequest, "could not load cache for %s/%s/%s/%s", false, collection, signature, action, paramstr)
		return
	}
}

func (mh *MediaHandler) SetRoutes(router *mux.Router) error {
	path := regexp.MustCompile(fmt.Sprintf("/%s/(?P<collection>[^/]+)/(?P<signature>[^/]+)/(?P<action>[^/]+)(/(?P<paramstr>.+))?$", mh.prefix))
	router.MatcherFunc(func(request *http.Request, match *mux.RouteMatch) bool {
		matches := path.FindStringSubmatch(request.URL.Path)
		if matches == nil {
			return false
		}
		match.Vars = map[string]string{}
		for i, name := range path.SubexpNames() {
			if name == "" {
				continue
			}
			match.Vars[name] = matches[i]
		}
		return true
	}).Methods("GET", "HEAD").Handler(mh)
	return nil
}

func (mh *MediaHandler) ingestMaster(collection, signature string) (*database.Master, *database.Cache, error) {
	coll, err := mh.mdb.GetCollectionByName(collection)
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot load collection %s", collection)
	}
	storage, err := coll.GetStorage()
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot load storage for %s", collection)
	}
	master, err := mh.mdb.GetMaster(coll, signature)
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot load master %s/%s", collection, signature)
	}
	// ingest master
	filename := storage.Filebase
	filename += "/" + filepath.Join(storage.DataDir, buildFilename(coll, master, "master", ""))
	reader, finfo, err := mh.FileOpenRead(master.Urn, filesystem.FileGetOptions{})
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot open master %s", master.Urn)
	}
	header, err := NewSideStream(mh.tempfolder, 2048)
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot create sidestream %s", master.Urn)
	}
	tempfile, err := header.Open()
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot open temp file in %s", mh.tempfolder)
	}
	defer header.Clear()
	newreader := io.TeeReader(reader, header)
	if err := mh.FileWrite(filename, newreader, finfo.Size(), filesystem.FilePutOptions{}); err != nil {
		reader.Close()
		return nil, nil, emperror.Wrapf(err, "cannot write cache/master %s", filename)
	}
	reader.Close()
	header.Close()
	var metadata = make(map[string]interface{})
	master.Type, master.Subtype, master.Mimetype, metadata, err = mh.idx.GetType(tempfile)
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot get type for %s", filename)
	}
	master.Sha256 = header.GetSHA256()

	width, height, duration, mimetype, sub, meta, err := mh.idx.GetMetadata(filename, master.Type, master.Subtype, master.Mimetype)
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot get metadata for %s", filename)
	}
	for key, val := range meta {
		metadata[key] = val
	}

	cache, err := database.NewCache(mh.mdb,
		0,
		coll.Id,
		master.Id,
		"master",
		"",
		mimetype,
		finfo.Size(),
		filename,
		width,
		height,
		duration,
	)
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot create cache %s", filename)
	}

	if err := cache.Store(); err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot store master cache for %s/%s", coll.Name, master.Signature)
	}

	master.Metadata = metadata
	master.Subtype = sub

	if err := master.Store(); err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot store master %s", master.Signature)
	}
	return master, cache, nil
}

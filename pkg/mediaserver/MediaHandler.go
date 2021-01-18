package mediaserver

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/goph/emperror"
	"github.com/gorilla/mux"
	"github.com/je4/zmedia/v2/pkg/database"
	"github.com/je4/zmedia/v2/pkg/filesystem"
	"github.com/je4/zmedia/v2/pkg/indexer"
	"github.com/op/go-logging"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type MediaHandler struct {
	log    *logging.Logger
	mdb    *database.MediaDatabase
	fss    map[string]filesystem.FileSystem
	prefix string
}

func buildFilename(coll *database.Collection, master *database.Master, action string, params []string) string {
	return fmt.Sprintf("%v.%v-%x", coll.Id, master.Id, md5.Sum([]byte(fmt.Sprintf("%s/%s/%s/%s", coll.Name, master.Signature, action, strings.Join(params, "/")))))
}

func NewMediaHandler(prefix string, mdb *database.MediaDatabase, log *logging.Logger, fss ...filesystem.FileSystem) (*MediaHandler, error) {
	mh := &MediaHandler{
		log:    log,
		prefix: prefix,
		mdb:    mdb,
		fss:    make(map[string]filesystem.FileSystem),
	}
	for _, fs := range fss {
		mh.fss[fs.Protocol()] = fs
	}
	return mh, nil
}

var pathRegexp = regexp.MustCompile(`^([^:]+://[^/]+)/([^/]+)/(.+)$`)

func (mh *MediaHandler) FileOpenRead(path string, opts filesystem.FileGetOptions) (io.ReadCloser, int64, error) {
	matches := pathRegexp.FindStringSubmatch(path)
	if matches == nil {
		return nil, 0, fmt.Errorf("invalid path - cannot load file %s from storage", path)
	}
	fs, ok := mh.fss[matches[1]]
	if !ok {
		return nil, 0, fmt.Errorf("invalid protocol - cannot find storage %s", matches[1])
	}
	return fs.FileOpenRead(matches[2], matches[3], opts)
}
func (mh *MediaHandler) FileWrite(path string, reader io.Reader, size int64, opts filesystem.FilePutOptions) error {
	matches := pathRegexp.FindStringSubmatch(path)
	if matches == nil {
		return fmt.Errorf("invalid path - cannot load file %s from storage", path)
	}
	fs, ok := mh.fss[matches[1]]
	if !ok {
		return fmt.Errorf("invalid protocol - cannot find storage %s", matches[1])
	}
	return fs.FileWrite(matches[2], matches[3], reader, size, opts)
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

func (mh *MediaHandler) WriteFile(resp io.Writer, path string) error {
	reader, _, err := mh.FileOpenRead(path, filesystem.FileGetOptions{})
	if err != nil {
		return fmt.Errorf("cannot open file %s: %v", path, err)
	}
	defer reader.Close()
	if _, err := io.Copy(resp, reader); err != nil {
		return fmt.Errorf("read file %s: %v", path, err)
	}
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
	filename := filepath.Join(storage.Filebase, storage.DataDir, buildFilename(coll, master, "master", []string{}))
	reader, size, err := mh.FileOpenRead(master.Urn, filesystem.FileGetOptions{})
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot open master %s", master.Urn)
	}
	var header indexer.HeaderMime
	newreader := io.TeeReader(reader, &header)
	if err := mh.FileWrite(filename, newreader, size, filesystem.FilePutOptions{}); err != nil {
		reader.Close()
		return nil, nil, emperror.Wrapf(err, "cannot write cache/master %s", filename)
	}
	reader.Close()
	mimetype := header.GetMime()
	cache, err := database.NewCache(mh.mdb,
		0,
		coll.Id,
		master.Id,
		"master",
		"",
		mimetype,
		size,
		filename,
		0,
		0,
		0,
	)
	if err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot create cache %s", filename)
	}
	// todo: identify

	if err := cache.Store(); err != nil {
		return nil, nil, emperror.Wrapf(err, "cannot store cache %s", filename)
	}
	return master, cache, nil
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

	cache, err := mh.mdb.GetCache(collection, signature, action, paramstr)
	// copy the file directly to the output
	if err == nil {
		resp.Header().Set("Content-type", cache.Mimetype)
		if err := mh.WriteFile(resp, cache.Path); err != nil {
			mh.DoPanicf(resp, http.StatusInternalServerError, err.Error(), false)
			return
		}
		return
	}
	if err == database.ErrNotFound {
		master, cache, err := mh.ingestMaster(collection, signature)
		if err != nil {
			mh.DoPanicf(resp, http.StatusInternalServerError, "cannot ingest master %s/%s: %v", false, collection, signature, err)
			return
		}
		mh.log.Infof("master: %v // %v", master, cache)
	}
	switch err {
	case database.ErrNotFound:

	case nil:
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

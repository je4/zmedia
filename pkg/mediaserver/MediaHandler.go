package mediaserver

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/je4/zmedia/v2/pkg/database"
	"github.com/je4/zmedia/v2/pkg/filesystem"
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
	fss map[string]filesystem.FileSystem
	prefix string
}

func NewMediaHandler(prefix string, mdb *database.MediaDatabase, fss []filesystem.FileSystem, log *logging.Logger) (*MediaHandler, error) {
	mh := &MediaHandler{
		log:    log,
		prefix: prefix,
		mdb:    mdb,
		fss: make(map[string]filesystem.FileSystem),
	}
	for _, fs := range fss {
		mh.fss[fs.Protocol()] = fs
	}
	return mh, nil
}

var errorTemplate = template.Must(template.New("error").Parse(`<html>
<head><title>{.Error}</title></head>
<body><h1>{.Error}</h1><h2>{.Message}</h2></body>
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

var pathRegexp = regexp.MustCompile(`^([^:]+://[^/]+)/([^/]+)/(.+)$`)

func (mh *MediaHandler) fileOpenRead(path string, opts filesystem.FileGetOptions) (io.ReadCloser, int64, error) {
	matches := pathRegexp.FindStringSubmatch(path)
	if matches == nil {
		return nil, 0, fmt.Errorf("invalid path - cannot load file %s from storage", path)
	}
	fs, ok :=  mh.fss[matches[1]]
	if !ok {
		return nil, 0, fmt.Errorf("invalid protocol - cannot find storage %s", matches[1])
	}
	return fs.FileOpenRead(matches[2], matches[3], opts)
}

func (mh *MediaHandler) fileWrite(path string, reader io.Reader, size int64, opts filesystem.FilePutOptions) error {
	matches := pathRegexp.FindStringSubmatch(path)
	if matches == nil {
		return fmt.Errorf("invalid path - cannot load file %s from storage", path)
	}
	fs, ok :=  mh.fss[matches[1]]
	if !ok {
		return fmt.Errorf("invalid protocol - cannot find storage %s", matches[1])
	}
	return fs.FileWrite(matches[2], matches[3], reader, size, opts)
}

func (mh *MediaHandler) WriteFile(resp io.Writer, path string) error {
	reader, _, err := mh.fileOpenRead(path, filesystem.FileGetOptions{})
	if err != nil {
		return fmt.Errorf("cannot open file %s: %v", path, err)
	}
	defer reader.Close()
	if _, err := io.Copy(resp, reader); err != nil {
		return fmt.Errorf("read file %s: %v", path, err)
	}
	return nil
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
		coll, err := mh.mdb.GetCollectionByName(collection)
		if err != nil {
			mh.DoPanicf(resp, http.StatusInternalServerError, "cannot load collection %s: %v", false, collection, err)
			return
		}
		storage, err := coll.GetStorage()
		if err != nil {
			mh.DoPanicf(resp, http.StatusInternalServerError, "cannot load storage for %s: %v", false, collection, err)
			return
		}
		master, err := mh.mdb.GetMaster(coll, signature)
		if err != nil {
			mh.DoPanicf(resp, http.StatusInternalServerError, "cannot load master %s/%s: %v", false, collection, signature, err)
			return
		}
		if action == "master" {
			// ingest master
			filename := filepath.Join(storage.DataDir, fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s/%s/%s", collection, signature, strings.Join(params, "/"))))))
			reader, size, err := mh.fileOpenRead(master.Urn, filesystem.FileGetOptions{})
			if err != nil {
				mh.DoPanicf(resp, http.StatusInternalServerError, "cannot open master %s: %v", false, master.Urn, err)
				return
			}
			if err := mh.fileWrite(filename, reader, size, filesystem.FilePutOptions{}); err != nil {
				reader.Close()
				mh.DoPanicf(resp, http.StatusInternalServerError, "cannot write cache/master %s: %v", false, filename, err)
				return
			}
			reader.Close()

			// identify
			now identify

		}
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

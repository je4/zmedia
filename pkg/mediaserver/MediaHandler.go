package mediaserver

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"regexp"
)

type MediaHandler struct {
	prefix string
}

func NewMediaHandler(prefix string) (*MediaHandler, error) {
	mh := &MediaHandler{prefix: prefix}
	return mh, nil
}

func (mh *MediaHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

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

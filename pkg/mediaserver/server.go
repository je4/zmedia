package mediaserver

import (
	"github.com/gorilla/mux"
	"net/http"
)

type MediaHandler struct{}

func (mh *MediaHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

}

func (mh *MediaHandler) SetRoutes(router *mux.Router) error {
	router.Methods("GET", "HEAD").PathPrefix("{collection}/{signature}/{action}").MatcherFunc(func(request *http.Request, match *mux.RouteMatch) bool {
		//match.Vars = map[string]string{}
		return true
	}).Handler(mh)
	return nil
}

func NewMediaHandler() (*MediaHandler, error) {
	mh := &MediaHandler{}
	return mh, nil
}

package web

import (
	"net/http"
	"slices"
)

// Router is the API's request router: Go's http.ServeMux plus a record of every pattern registered on
// it, so that a test can check that the API serves exactly the operations of its contract (research R11).
type Router struct {
	mux      *http.ServeMux
	patterns []string
}

// NewRouter returns a router without routes.
func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

// Handle registers h for pattern, written as an http.ServeMux pattern: "PATCH /api/sample-folders/{folderId}".
// Registering the same pattern twice panics when the API starts, which is how ServeMux reports the mistake.
func (r *Router) Handle(pattern string, h http.HandlerFunc) {
	r.mux.Handle(pattern, h)
	r.patterns = append(r.patterns, pattern)
}

// Patterns returns the registered patterns in the order they were registered.
func (r *Router) Patterns() []string {
	return slices.Clone(r.patterns) // a copy, so the caller cannot change the router's record
}

// ServeHTTP sends the request to the handler whose pattern matches it.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

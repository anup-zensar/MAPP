package web

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Router serves net/http handlers of packages not yet on Gin, filling in r.Pattern and r.PathValue.
type Router struct {
	routes gin.IRoutes
}

func NewRouter(routes gin.IRoutes) *Router {
	return &Router{routes: routes}
}

// Handle registers h for a ServeMux pattern such as "PATCH /api/v1/sample-folders/{folderId}".
func (r *Router) Handle(pattern string, h http.HandlerFunc) {
	method, path, _ := strings.Cut(pattern, " ")
	r.routes.Handle(method, ginPath(path), func(c *gin.Context) {
		c.Request.Pattern = pattern
		for _, parameter := range c.Params {
			c.Request.SetPathValue(parameter.Key, parameter.Value)
		}
		h(c.Writer, c.Request)
	})
}

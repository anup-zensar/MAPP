package web

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/errs"
)

// BindJSON decodes the body into dst, or returns bad_request for anything but one strict JSON value.
func BindJSON(c *gin.Context, dst any) error {
	return readBody(c.Request, c.Request.Body, dst)
}

func Respond(c *gin.Context, status int, body any) {
	code, encoded, err := encode(status, body)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "could not encode the answer", "error", err)
	}
	c.Data(code, jsonType, encoded)
}

// RespondError answers an *errs.Error as it is and logs any other error as internal_error, then aborts.
func RespondError(c *gin.Context, err error) {
	apiErr, ok := errors.AsType[*errs.Error](err)
	if !ok {
		slog.ErrorContext(c.Request.Context(), "request failed", "method", c.Request.Method,
			"path", c.Request.URL.Path, "error", err)
		apiErr = errs.Internal()
	}
	Respond(c, apiErr.Status, errorEnvelope{Error: apiErr})
	c.Abort()
}

// Operation is the request's key in the route table and the contract: "GET /api/v1/plans/{planId}".
func Operation(c *gin.Context) string {
	return c.Request.Method + " " + contractPath(c.FullPath())
}

func Operations(routes gin.RoutesInfo) []string {
	operations := make([]string, 0, len(routes))
	for _, route := range routes {
		operations = append(operations, route.Method+" "+contractPath(route.Path))
	}
	return operations
}

func contractPath(ginPath string) string {
	segments := strings.Split(ginPath, "/")
	for i, segment := range segments {
		if name, isParameter := strings.CutPrefix(segment, ":"); isParameter {
			segments[i] = "{" + name + "}"
		}
	}
	return strings.Join(segments, "/")
}

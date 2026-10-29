package web

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/logging"
	"mapp/internal/platform/querycount"
)

const requestIDHeader = "X-Request-ID"

// RequestID ignores any ID the client sends, so a client cannot write into the logs.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := rand.Text()
		c.Header(requestIDHeader, id)
		c.Request = c.Request.WithContext(logging.WithAttrs(c.Request.Context(), slog.String("request_id", id)))
		c.Next()
	}
}

func LogRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		counted := countStatements(c)
		c.Next()
		slog.InfoContext(c.Request.Context(), "request", "method", c.Request.Method, "path", c.Request.URL.Path,
			"route", c.FullPath(), "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds(),
			"queries", querycount.Of(counted))
	}
}

// countStatements gives the request a statement counter and returns the context that holds it.
func countStatements(c *gin.Context) context.Context {
	counted := querycount.With(c.Request.Context())
	c.Request = c.Request.WithContext(counted)
	return counted
}

// Recover answers a panic with internal_error unless the answer has started; a nil writer keeps Gin from
// dumping the request, session cookie included.
func Recover() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		slog.ErrorContext(c.Request.Context(), "panic in handler", "panic", recovered, "stack", string(debug.Stack()))
		if c.Writer.Written() {
			c.Abort()
			return
		}
		RespondError(c, errs.Internal())
	})
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// SameOrigin refuses other hosts (DNS rebinding) and browser requests from other sites.
func SameOrigin() gin.HandlerFunc {
	crossOrigin := http.NewCrossOriginProtection()
	return func(c *gin.Context) {
		r := c.Request
		if !isLocalHost(r.Host) || crossOrigin.Check(r) != nil || foreignReferer(r) {
			RespondError(c, errs.BadRequest(otherSiteMessage))
			return
		}
		c.Next()
	}
}

// foreignReferer covers clients that send neither Origin nor Sec-Fetch-Site, which the standard check lets through.
func foreignReferer(r *http.Request) bool {
	referer := r.Header.Get("Referer")
	if isSafeMethod(r.Method) || referer == "" || r.Header.Get("Origin") != "" || r.Header.Get("Sec-Fetch-Site") != "" {
		return false
	}
	parsed, err := url.Parse(referer)
	return err != nil || !strings.EqualFold(parsed.Host, r.Host)
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// RejectControlInPath answers not_found when a path parameter holds a control character; Gin has
// matched the route by then, so the parameters are known.
func RejectControlInPath() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, param := range c.Params {
			if hasControl(param.Value) {
				RespondError(c, errs.NotFound(notFoundMessage))
				return
			}
		}
		c.Next()
	}
}

// LimitBody caps every body at defaultMax bytes, or at the limit byOperation gives its operation.
func LimitBody(defaultMax int64, byOperation map[string]int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, found := byOperation[Operation(c)]
		if !found {
			limit = defaultMax
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func isLocalHost(hostAndPort string) bool {
	host := hostAndPort
	if name, _, err := net.SplitHostPort(hostAndPort); err == nil {
		host = name
	}
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	switch strings.ToLower(host) {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
}

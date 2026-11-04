package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger records every HTTP request and includes errors attached by
// handlers. Request bodies and headers are deliberately excluded because they
// may contain bearer tokens or device credentials.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()

		status := c.Writer.Status()
		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"route", c.FullPath(),
			"status", status,
			"latency_ms", time.Since(started).Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if userID := UserID(c); userID != "" {
			attrs = append(attrs, "user_id", userID)
		}
		if len(c.Errors) > 0 {
			errs := make([]string, 0, len(c.Errors))
			for _, err := range c.Errors {
				errs = append(errs, err.Err.Error())
			}
			attrs = append(attrs, "errors", errs)
		}

		switch {
		case status >= http.StatusInternalServerError:
			logger.Error("request failed", attrs...)
		case status >= http.StatusBadRequest:
			logger.Warn("request rejected", attrs...)
		default:
			logger.Info("request completed", attrs...)
		}
	}
}

// UseSlogForGinDebug sends Gin's own debug output (route registrations and
// debug-mode warnings), which Gin otherwise prints as plain text to stdout,
// through logger so every server log line is structured. Gin only produces
// this output in debug mode, i.e. outside production.
func UseSlogForGinDebug(logger *slog.Logger) {
	gin.DebugPrintRouteFunc = func(method, path, handler string, handlers int) {
		logger.Debug("route registered", "component", "gin", "method", method, "path", path, "handler", handler, "handlers", handlers)
	}
	gin.DebugPrintFunc = func(format string, values ...any) {
		msg := strings.TrimSpace(fmt.Sprintf(format, values...))
		if warning, ok := strings.CutPrefix(msg, "[WARNING] "); ok {
			logger.Warn(warning, "component", "gin")
			return
		}
		logger.Debug(msg, "component", "gin")
	}
}

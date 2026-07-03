// Package middleware holds cross-cutting Gin middleware shared across domains.
package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

// CORS returns middleware that allows browser requests from the given origins.
//
// The request's Origin is echoed back only when it appears in allowedOrigins,
// so the allow-list stays explicit. A single "*" entry allows any origin (do not
// combine it with credentials). Preflight OPTIONS requests are answered here with
// 204 and never reach the route handlers.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := slices.Contains(allowedOrigins, "*")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" && (allowAll || slices.Contains(allowedOrigins, origin)) {
			if allowAll {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
				// Responses vary per Origin, so caches must key on it.
				c.Header("Vary", "Origin")
			}
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

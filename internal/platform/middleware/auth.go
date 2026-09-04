package middleware

import (
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/gin-gonic/gin"
)

const userIDKey = "clerk_user_id"

// ClerkAuth verifies Clerk bearer session tokens and stores their subject in Gin context.
func ClerkAuth(secretKey string, authorizedParties []string) gin.HandlerFunc {
	clerk.SetKey(secretKey)
	allowed := make(map[string]struct{}, len(authorizedParties))
	for _, party := range authorizedParties {
		allowed[party] = struct{}{}
	}
	failure := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Unauthorized"}`))
	})
	verify := clerkhttp.WithHeaderAuthorization(
		clerkhttp.AuthorizedParty(func(party string) bool { _, ok := allowed[party]; return party != "" && ok }),
		clerkhttp.AuthorizationFailureHandler(failure),
	)

	return func(c *gin.Context) {
		verified := false
		verify(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			claims, ok := clerk.SessionClaimsFromContext(r.Context())
			if !ok || claims == nil || claims.Subject == "" {
				return
			}
			verified = true
			c.Request = r
			c.Set(userIDKey, claims.Subject)
		})).ServeHTTP(c.Writer, c.Request)
		if !verified {
			if !c.Writer.Written() {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			}
			c.Abort()
			return
		}
		c.Next()
	}
}

func UserID(c *gin.Context) string { return c.GetString(userIDKey) }

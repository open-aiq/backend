package middleware

import (
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/gin-gonic/gin"
	"go-aiq-backend/internal/platform/problem"
)

const (
	userIDKey       = "clerk_user_id"
	nativeClientKey = "clerk_native_client"
)

// ClerkAuth verifies Clerk bearer session tokens and stores their subject in Gin context.
//
// Browser tokens must carry an authorized-party (azp) claim from
// authorizedParties. Tokens issued to native clients (the mobile app) carry no
// azp at all; they are accepted only when allowNativeClients is true.
func ClerkAuth(secretKey string, authorizedParties []string, allowNativeClients bool) gin.HandlerFunc {
	clerk.SetKey(secretKey)
	allowed := make(map[string]struct{}, len(authorizedParties))
	for _, party := range authorizedParties {
		allowed[party] = struct{}{}
	}
	failure := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problem.WriteHTTP(w, r, problem.Unauthorized, "A valid Clerk session token is required.")
	})
	verify := clerkhttp.WithHeaderAuthorization(
		clerkhttp.AuthorizedParty(func(party string) bool { return authorizedParty(allowed, allowNativeClients, party) }),
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
			c.Set(nativeClientKey, claims.AuthorizedParty == "")
		})).ServeHTTP(c.Writer, c.Request)
		if !verified {
			if !c.Writer.Written() {
				problem.Write(c, problem.Unauthorized, "A valid Clerk session token is required.")
			}
			c.Abort()
			return
		}
		c.Next()
	}
}

func UserID(c *gin.Context) string { return c.GetString(userIDKey) }

// NativeClient reports whether the request's session token was issued to a
// native client (the mobile app). Such tokens carry no authorized party.
func NativeClient(c *gin.Context) bool { return c.GetBool(nativeClientKey) }

// RequireNativeClient rejects requests made with browser session tokens. Use it
// after ClerkAuth for actions that only the mobile app can complete.
func RequireNativeClient() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !NativeClient(c) {
			problem.Write(c, problem.MobileOnly, "Add monitors and replace device keys from the Open AIQ mobile app.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func authorizedParty(allowed map[string]struct{}, allowNativeClients bool, party string) bool {
	if party == "" {
		return allowNativeClients
	}
	_, ok := allowed[party]
	return ok
}

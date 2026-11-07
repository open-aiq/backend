package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthorizedParty(t *testing.T) {
	allowed := map[string]struct{}{"https://openaiq.org": {}}
	cases := []struct {
		name   string
		native bool
		party  string
		want   bool
	}{
		{"listed origin", false, "https://openaiq.org", true},
		{"unlisted origin", true, "https://evil.example", false},
		{"native token rejected by default", false, "", false},
		{"native token accepted when enabled", true, "", true},
	}
	for _, tc := range cases {
		if got := authorizedParty(allowed, tc.native, tc.party); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestRequireNativeClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name   string
		native *bool
		want   int
	}{
		{"native token passes", ptr(true), http.StatusNoContent},
		{"browser token rejected", ptr(false), http.StatusForbidden},
		{"missing flag rejected", nil, http.StatusForbidden},
	}
	for _, tc := range cases {
		r := gin.New()
		r.POST("/devices", func(c *gin.Context) {
			if tc.native != nil {
				c.Set(nativeClientKey, *tc.native)
			}
			c.Next()
		}, RequireNativeClient(), func(c *gin.Context) { c.Status(http.StatusNoContent) })

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/devices", nil))
		if w.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, w.Code, tc.want)
		}
		if tc.want == http.StatusForbidden && !strings.Contains(w.Body.String(), "mobile-app-required") {
			t.Errorf("%s: body %q lacks mobile-app-required", tc.name, w.Body.String())
		}
	}
}

func ptr[T any](v T) *T { return &v }

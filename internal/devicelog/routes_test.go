package devicelog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// headerAuthenticator maps each X-Device-ID to a stable internal id.
type headerAuthenticator struct{ ids map[string]uuid.UUID }

func (a headerAuthenticator) AuthenticateDevice(_ context.Context, deviceID, _, _ string) (uuid.UUID, error) {
	if _, ok := a.ids[deviceID]; !ok {
		a.ids[deviceID] = uuid.New()
	}
	return a.ids[deviceID], nil
}

func TestUploadRouteRateLimitsPerDevice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(NewService(&repoStub{})).RegisterDeviceRoutes(r.Group("/api/v1"), headerAuthenticator{ids: map[string]uuid.UUID{}})

	upload := func(device string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/device/logs", strings.NewReader(`{"entries":[]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Device-ID", device)
		req.Header.Set("X-Device-Key", "sk_test")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	for i := range UploadsPerWindow {
		if w := upload("dev_a"); w.Code == http.StatusTooManyRequests {
			t.Fatalf("upload %d was rate limited", i)
		}
	}
	w := upload("dev_a")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header missing")
	}
	if w := upload("dev_b"); w.Code == http.StatusTooManyRequests {
		t.Fatal("other devices must not be limited")
	}
}

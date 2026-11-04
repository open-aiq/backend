package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type authenticatorStub struct {
	id      uuid.UUID
	err     error
	version string
}

func (a *authenticatorStub) AuthenticateDevice(_ context.Context, _, _, version string) (uuid.UUID, error) {
	a.version = version
	return a.id, a.err
}

func serveDeviceAuth(t *testing.T, auth DeviceAuthenticator, headers map[string]string) (*httptest.ResponseRecorder, uuid.UUID) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var seen uuid.UUID
	r.GET("/x", DeviceAuth(auth), func(c *gin.Context) {
		seen = DeviceID(c)
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, seen
}

func TestDeviceAuthRequiresHeaders(t *testing.T) {
	w, _ := serveDeviceAuth(t, &authenticatorStub{}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestDeviceAuthRejectsInvalidCredentials(t *testing.T) {
	auth := &authenticatorStub{err: ErrInvalidDeviceCredentials}
	w, _ := serveDeviceAuth(t, auth, map[string]string{HeaderDeviceID: "dev_1", HeaderDeviceKey: "sk_1"})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestDeviceAuthReportsInternalErrors(t *testing.T) {
	auth := &authenticatorStub{err: errors.New("db down")}
	w, _ := serveDeviceAuth(t, auth, map[string]string{HeaderDeviceID: "dev_1", HeaderDeviceKey: "sk_1"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestDeviceAuthStoresDeviceAndNormalizesFirmwareVersion(t *testing.T) {
	id := uuid.New()
	auth := &authenticatorStub{id: id}
	w, seen := serveDeviceAuth(t, auth, map[string]string{
		HeaderDeviceID: "dev_1", HeaderDeviceKey: "sk_1", HeaderFirmwareVersion: "v0.2.0",
	})
	if w.Code != http.StatusNoContent || seen != id {
		t.Fatalf("status = %d, id = %s", w.Code, seen)
	}
	if auth.version != "0.2.0" {
		t.Fatalf("version = %q", auth.version)
	}
}

func TestNormalizeFirmwareVersion(t *testing.T) {
	cases := map[string]string{
		"0.1.0":       "0.1.0",
		" v1.2.3 ":    "1.2.3",
		"1.4.0-rc.1":  "1.4.0-rc.1",
		"":            "",
		"latest":      "",
		"1.2":         "",
		"1.2.3; DROP": "",
		"99999.0.0":   "",
	}
	for input, want := range cases {
		if got := NormalizeFirmwareVersion(input); got != want {
			t.Errorf("NormalizeFirmwareVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestStaticBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/p", StaticBearer("secret-token"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for header, want := range map[string]int{
		"":                    http.StatusUnauthorized,
		"Bearer wrong":        http.StatusUnauthorized,
		"secret-token":        http.StatusUnauthorized,
		"Bearer secret-token": http.StatusNoContent,
	} {
		req := httptest.NewRequest(http.MethodPost, "/p", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != want {
			t.Errorf("Authorization %q: status = %d, want %d", header, w.Code, want)
		}
	}
}

package middleware

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestLoggerIncludesHandlerErrorWithoutSensitiveHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := gin.New()
	router.Use(RequestLogger(logger))
	router.POST("/api/v1/devices", func(c *gin.Context) {
		_ = c.Error(errors.New("create device: database unavailable"))
		c.Status(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", nil)
	req.Header.Set("Authorization", "Bearer secret-session-token")
	req.Header.Set("X-Device-Key", "sk_secret-device-key")
	router.ServeHTTP(httptest.NewRecorder(), req)

	logLine := output.String()
	for _, want := range []string{
		`"level":"ERROR"`,
		`"route":"/api/v1/devices"`,
		`"status":500`,
		`create device: database unavailable`,
	} {
		if !strings.Contains(logLine, want) {
			t.Fatalf("log output %q does not contain %q", logLine, want)
		}
	}
	for _, secret := range []string{"secret-session-token", "secret-device-key"} {
		if strings.Contains(logLine, secret) {
			t.Fatalf("log output contains sensitive value %q", secret)
		}
	}
}

func TestUseSlogForGinDebugWritesRoutesAndWarningsAsJSON(t *testing.T) {
	routeFunc, printFunc := gin.DebugPrintRouteFunc, gin.DebugPrintFunc
	t.Cleanup(func() {
		gin.DebugPrintRouteFunc, gin.DebugPrintFunc = routeFunc, printFunc
		gin.SetMode(gin.TestMode)
	})

	var output bytes.Buffer
	UseSlogForGinDebug(slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	gin.SetMode(gin.DebugMode)
	router := gin.New()
	router.GET("/ping", func(c *gin.Context) {})

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "{") {
			t.Fatalf("gin debug output is not JSON: %q", line)
		}
	}
	for _, want := range []string{
		`"level":"WARN","msg":"Running in \"debug\" mode.`,
		`"msg":"route registered","component":"gin","method":"GET","path":"/ping"`,
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("log output %q does not contain %q", output.String(), want)
		}
	}
}

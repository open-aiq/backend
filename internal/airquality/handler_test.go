package airquality

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func customRange(t *testing.T, start, end string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/custom", NewHandler(NewService(mapRepositoryStub{})).GetCustomRange)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/custom?start_date="+start+"&end_date="+end, nil))
	return w
}

func TestCustomRangeAcceptsUpToMaxDays(t *testing.T) {
	// 2024 is a leap year: 1 Jan to 31 Dec is exactly 366 days.
	if w := customRange(t, "2024-01-01", "2024-12-31"); w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestCustomRangeRejectsLongerRanges(t *testing.T) {
	w := customRange(t, "2024-01-01", "2025-01-01")
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	for _, want := range []string{`"name":"end_date"`, `"code":"max_range"`, "within 366 days"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("body %s does not contain %s", w.Body.String(), want)
		}
	}
}

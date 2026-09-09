package problem

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWriteUsesRFC9457Contract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/devices?secret=no", nil)
	Write(c, Validation, "Invalid input.", Violation{In: "body", Name: "name", Code: "required", Detail: "name is required"})
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("content type = %q", got)
	}
	if strings.Contains(recorder.Body.String(), "secret=no") {
		t.Fatal("instance leaked query parameters")
	}
}

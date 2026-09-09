package input

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go-aiq-backend/internal/platform/problem"
)

type testPayload struct {
	Name string `json:"name" mod:"trim" validate:"required,max=10"`
}

func TestBindJSONTransformsAndValidates(t *testing.T) {
	c, recorder := contextFor(http.MethodPost, "/items", `{"name":"  Sensor  "}`, "application/json; charset=utf-8")
	var payload testPayload
	if !BindJSON(c, &payload) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
	if payload.Name != "Sensor" {
		t.Fatalf("Name = %q", payload.Name)
	}
}

func TestBindJSONFailures(t *testing.T) {
	tests := []struct {
		name, body, contentType string
		status                  int
		code                    string
	}{
		{"empty", "", "application/json", 400, ""},
		{"malformed", "{", "application/json", 400, ""},
		{"trailing", `{"name":"ok"} {}`, "application/json", 400, ""},
		{"unknown", `{"name":"ok","extra":true}`, "application/json", 422, "unknown"},
		{"wrong type", `{"name":4}`, "application/json", 422, "invalid_type"},
		{"blank after trim", `{"name":"   "}`, "application/json", 422, "required"},
		{"media type", `{"name":"ok"}`, "text/plain", 415, ""},
		{"too large", `{"name":"` + strings.Repeat("a", int(MaxJSONBodyBytes)) + `"}`, "application/json", 413, ""},
		{"too much trailing whitespace", `{"name":"ok"}` + strings.Repeat(" ", int(MaxJSONBodyBytes)), "application/json", 413, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder := contextFor(http.MethodPost, "/items", tt.body, tt.contentType)
			var payload testPayload
			if BindJSON(c, &payload) {
				t.Fatal("expected failure")
			}
			if recorder.Code != tt.status {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
				t.Fatalf("content type = %q", got)
			}
			var body problem.Problem
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Status != tt.status || body.Instance != "/items" {
				t.Fatalf("problem = %+v", body)
			}
			if tt.code != "" && (len(body.Errors) != 1 || body.Errors[0].Code != tt.code) {
				t.Fatalf("errors = %+v", body.Errors)
			}
		})
	}
}

func TestValidateTransformsPointerFieldsBeforeValidation(t *testing.T) {
	name := "  Balcony  "
	payload := struct {
		Name *string `json:"name" mod:"trim" validate:"omitempty,min=1"`
	}{Name: &name}
	c, recorder := contextFor(http.MethodPatch, "/items/1", "", "")
	if !Validate(c, "body", &payload) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
	if got := *payload.Name; got != "Balcony" {
		t.Fatalf("Name = %q", got)
	}

	blank := "   "
	payload.Name = &blank
	c, recorder = contextFor(http.MethodPatch, "/items/1", "", "")
	if Validate(c, "body", &payload) {
		t.Fatal("expected a provided blank value to fail validation")
	}
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRejectUnknownQuery(t *testing.T) {
	c, recorder := contextFor(http.MethodGet, "/items?timeline=daily&extra=1", "", "")
	if RejectUnknownQuery(c, "timeline") {
		t.Fatal("expected unknown query failure")
	}
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func contextFor(method, target, body, contentType string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}
	return c, recorder
}

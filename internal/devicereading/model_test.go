package devicereading

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go-aiq-backend/internal/platform/input"
)

func TestUploadRequestAcceptsZeroReadingsAndNormalizesProviders(t *testing.T) {
	body := `{
		"pms_data":{"pm1_0":0,"pm2_5":0,"pm10_0":0,"provider":"  pms5003  "},
		"aqi":0,
		"temperature_data":{"temperature":0,"humidity":0,"heat_index":0,"provider":" dht22 "}
	}`
	c, recorder := readingContext(body)
	var request UploadRequest
	if !input.BindJSON(c, &request) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
	if request.PMSData.Provider != "pms5003" || request.TemperatureData.Provider != "dht22" {
		t.Fatalf("providers were not normalized: %+v", request)
	}
}

func TestUploadRequestRejectsInvalidNestedValues(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"humidity", `{"pms_data":{"pm1_0":1,"pm2_5":1,"pm10_0":1,"provider":"pms5003"},"aqi":1,"temperature_data":{"temperature":20,"humidity":101,"heat_index":20,"provider":"dht22"}}`},
		{"partial location", `{"pms_data":{"pm1_0":1,"pm2_5":1,"pm10_0":1,"provider":"pms5003"},"aqi":1,"temperature_data":{"temperature":20,"humidity":50,"heat_index":20,"provider":"dht22"},"location":{"lat":24,"provider":"mobile"}}`},
		{"provider casing", `{"pms_data":{"pm1_0":1,"pm2_5":1,"pm10_0":1,"provider":"PMS5003"},"aqi":1,"temperature_data":{"temperature":20,"humidity":50,"heat_index":20,"provider":"dht22"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder := readingContext(tt.body)
			var request UploadRequest
			if input.BindJSON(c, &request) {
				t.Fatal("expected validation failure")
			}
			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func readingContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/data", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, recorder
}

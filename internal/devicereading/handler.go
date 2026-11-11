package devicereading

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-aiq-backend/internal/platform/input"
	"go-aiq-backend/internal/platform/middleware"
	"go-aiq-backend/internal/platform/problem"
)

// Handler holds dependencies for device reading HTTP handlers.
type Handler struct {
	service *Service
	auth    middleware.DeviceAuthenticator
}

// NewHandler creates a new device reading handler. auth authenticates uploads
// (implemented by device.Service; wired in main.go).
func NewHandler(service *Service, auth middleware.DeviceAuthenticator) *Handler {
	return &Handler{service: service, auth: auth}
}

// Upload godoc
//
// @Summary Upload a sensor reading
// @Description Stores one full sensor reading (particulate matter, AQI, temperature, optional location). The device authenticates with its X-Device-ID and X-Device-Key headers. Limited to 30 uploads per device per minute.
// @Tags Device Data
// @Accept json
// @Produce json
//
// @Param X-Device-ID header string true "Public device id (dev_...)"
// @Param X-Device-Key header string true "Secret device key (sk_...)"
// @Param X-Firmware-Version header string false "Firmware version running on the device, e.g. 0.2.0"
// @Param request body UploadRequest true "Sensor reading"
//
// @Success 201 {object} UploadResponse
// @Failure 400 {object} problem.Problem
// @Failure 401 {object} problem.Problem
// @Failure 413 {object} problem.Problem
// @Failure 415 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 429 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Router /data [post]
func (h *Handler) Upload(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	var req UploadRequest
	if !input.BindJSON(c, &req) {
		return
	}

	deviceID := middleware.DeviceID(c)

	created, err := h.service.Ingest(c.Request.Context(), deviceID, &req)
	if err != nil {
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to store reading.")
		return
	}

	c.IndentedJSON(http.StatusCreated, UploadResponse{Data: *created})
}

package devicereading

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"go-aiq-backend/internal/device"
)

// Header names a device authenticates with, and the context key the middleware
// stores the authenticated device's internal id under.
const (
	headerDeviceID  = "X-Device-ID"
	headerDeviceKey = "X-Device-Key"
	ctxDeviceID     = "devicereading.device_id"
)

// Authenticator verifies device credentials and returns the device's internal
// id. Implemented by device.Service; wired in main.go.
type Authenticator interface {
	Authenticate(ctx context.Context, deviceID, deviceKey string) (uuid.UUID, error)
}

// Handler holds dependencies for device reading HTTP handlers.
type Handler struct {
	service *Service
	auth    Authenticator
}

// NewHandler creates a new device reading handler.
func NewHandler(service *Service, auth Authenticator) *Handler {
	return &Handler{service: service, auth: auth}
}

// deviceAuth authenticates the request via the X-Device-ID / X-Device-Key
// headers and stores the device's internal id in the context.
func (h *Handler) deviceAuth(c *gin.Context) {
	deviceID := c.GetHeader(headerDeviceID)
	deviceKey := c.GetHeader(headerDeviceKey)
	if deviceID == "" || deviceKey == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Missing device credentials",
			Details: headerDeviceID + " and " + headerDeviceKey + " headers are required",
		})
		return
	}

	id, err := h.auth.Authenticate(c.Request.Context(), deviceID, deviceKey)
	if err != nil {
		if errors.Is(err, device.ErrInvalidCredentials) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid device credentials"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to authenticate device"})
		return
	}

	c.Set(ctxDeviceID, id)
	c.Next()
}

// Upload godoc
//
// @Summary Upload a sensor reading
// @Description Stores one full sensor reading (particulate matter, AQI, temperature, optional location). The device authenticates with its X-Device-ID and X-Device-Key headers.
// @Tags Device Data
// @Accept json
// @Produce json
//
// @Param X-Device-ID header string true "Public device id (dev_...)"
// @Param X-Device-Key header string true "Secret device key (sk_...)"
// @Param request body UploadRequest true "Sensor reading"
//
// @Success 201 {object} UploadResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /data [post]
func (h *Handler) Upload(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	deviceID := c.MustGet(ctxDeviceID).(uuid.UUID)

	created, err := h.service.Ingest(c.Request.Context(), deviceID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to store reading"})
		return
	}

	c.IndentedJSON(http.StatusCreated, UploadResponse{Data: *created})
}

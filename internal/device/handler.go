package device

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler holds dependencies for device HTTP handlers.
type Handler struct {
	service *Service
}

// NewHandler creates a new device handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create godoc
//
// @Summary Register a new device
// @Description Registers a device and returns its credentials. The device_key is a secret shown only once here; store it securely.
// @Tags Devices
// @Accept json
// @Produce json
//
// @Param request body CreateDeviceRequest true "Device to register"
//
// @Success 201 {object} CreateDeviceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /devices [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateDeviceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: "name is required",
		})
		return
	}

	created, err := h.service.Create(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create device"})
		return
	}

	c.IndentedJSON(http.StatusCreated, CreateDeviceResponse{Data: *created})
}

// List godoc
//
// @Summary List all devices
// @Description Returns all registered devices. Secret keys are never included.
// @Tags Devices
// @Produce json
//
// @Success 200 {object} ListDevicesResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /devices [get]
func (h *Handler) List(c *gin.Context) {
	devices, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to list devices"})
		return
	}

	c.IndentedJSON(http.StatusOK, ListDevicesResponse{Data: devices})
}

// Delete godoc
//
// @Summary Delete a device
// @Description Permanently deletes a device by its id.
// @Tags Devices
// @Produce json
//
// @Param id path string true "Device id (UUID)"
//
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /devices/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid device id",
			Details: "id must be a UUID",
		})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Device not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete device"})
		return
	}

	c.Status(http.StatusNoContent)
}

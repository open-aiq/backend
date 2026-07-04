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

	created, err := h.service.Create(c.Request.Context(), req)
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

// Update godoc
//
// @Summary Update a device
// @Description Partially updates a device: only the fields present in the body (name, is_outdoor, is_public) are changed.
// @Tags Devices
// @Accept json
// @Produce json
//
// @Param id path string true "Device id (UUID)"
// @Param request body UpdateDeviceRequest true "Fields to update"
//
// @Success 200 {object} UpdateDeviceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /devices/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid device id",
			Details: "id must be a UUID",
		})
		return
	}

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	updated, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoUpdateFields):
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "No fields to update",
				Details: "provide at least one of: name, is_outdoor, is_public",
			})
		case errors.Is(err, ErrDeviceNotFound):
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Device not found"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update device"})
		}
		return
	}

	c.IndentedJSON(http.StatusOK, UpdateDeviceResponse{Data: *updated})
}

// RotateKey godoc
//
// @Summary Rotate a device's secret key
// @Description Generates a new device_key for the device, invalidating the old one immediately. The new key is shown only in this response; update the physical device with it right away.
// @Tags Devices
// @Produce json
//
// @Param id path string true "Device id (UUID)"
//
// @Success 200 {object} RotateKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /devices/{id}/rotate-key [post]
func (h *Handler) RotateKey(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid device id",
			Details: "id must be a UUID",
		})
		return
	}

	rotated, err := h.service.RotateKey(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Device not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to rotate device key"})
		return
	}

	c.IndentedJSON(http.StatusOK, RotateKeyResponse{Data: *rotated})
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

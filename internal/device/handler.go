package device

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-aiq-backend/internal/platform/input"
	"go-aiq-backend/internal/platform/middleware"
	"go-aiq-backend/internal/platform/problem"
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
// @Failure 401 {object} problem.Problem
// @Failure 400 {object} problem.Problem
// @Failure 413 {object} problem.Problem
// @Failure 415 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Security BearerAuth
// @Router /devices [post]
func (h *Handler) Create(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	var req CreateDeviceRequest

	if !input.BindJSON(c, &req) {
		return
	}

	created, err := h.service.Create(c.Request.Context(), middleware.UserID(c), req)
	if err != nil {
		if errors.Is(err, ErrLocationRequiresPublic) {
			problem.Write(c, problem.Validation, "Location sharing requires a public device.", problem.Violation{In: "body", Name: "is_location_public", Code: "requires", Detail: "is_location_public requires is_public to be true"})
			return
		}
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to create device.")
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
// @Failure 401 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Security BearerAuth
// @Router /devices [get]
func (h *Handler) List(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	devices, err := h.service.List(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to list devices.")
		return
	}

	c.IndentedJSON(http.StatusOK, ListDevicesResponse{Data: devices})
}

// ListPublic godoc
// @Summary List public devices
// @Tags Public
// @Produce json
// @Success 200 {object} ListPublicDevicesResponse
// @Router /public/devices [get]
func (h *Handler) ListPublic(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	devices, err := h.service.ListPublic(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to list public devices.")
		return
	}
	c.IndentedJSON(http.StatusOK, ListPublicDevicesResponse{Data: devices})
}

// GetPublic godoc
// @Summary Get a public device
// @Tags Public
// @Produce json
// @Param id path string true "Device id (UUID)"
// @Success 200 {object} PublicDeviceResponse
// @Failure 422 {object} problem.Problem
// @Failure 404 {object} problem.Problem
// @Router /public/devices/{id} [get]
func (h *Handler) GetPublic(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	id, ok := deviceID(c)
	if !ok {
		return
	}
	publicDevice, err := h.service.GetPublic(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			problem.Write(c, problem.NotFound, "Device not found.")
			return
		}
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to get public device.")
		return
	}
	c.IndentedJSON(http.StatusOK, PublicDeviceResponse{Data: *publicDevice})
}

// Update godoc
//
// @Summary Update a device
// @Description Partially updates a device. Exact location can only be shared by a public device, and making a device private revokes location sharing.
// @Tags Devices
// @Accept json
// @Produce json
//
// @Param id path string true "Device id (UUID)"
// @Param request body UpdateDeviceRequest true "Fields to update"
//
// @Success 200 {object} UpdateDeviceResponse
// @Failure 401 {object} problem.Problem
// @Failure 400 {object} problem.Problem
// @Failure 404 {object} problem.Problem
// @Failure 413 {object} problem.Problem
// @Failure 415 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Security BearerAuth
// @Router /devices/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	id, ok := deviceID(c)
	if !ok {
		return
	}

	var req UpdateDeviceRequest
	if !input.BindJSON(c, &req) {
		return
	}

	updated, err := h.service.Update(c.Request.Context(), middleware.UserID(c), id, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoUpdateFields):
			problem.Write(c, problem.Validation, "Provide at least one field to update.", problem.Violation{In: "body", Name: "request", Code: "required", Detail: "provide at least one of: name, is_outdoor, is_public, is_location_public"})
		case errors.Is(err, ErrLocationRequiresPublic):
			problem.Write(c, problem.Validation, "Location sharing requires a public device.", problem.Violation{In: "body", Name: "is_location_public", Code: "requires", Detail: "is_location_public requires is_public to be true"})
		case errors.Is(err, ErrDeviceNotFound):
			problem.Write(c, problem.NotFound, "Device not found.")
		default:
			_ = c.Error(err)
			problem.Write(c, problem.Internal, "Failed to update device.")
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
// @Failure 401 {object} problem.Problem
// @Failure 404 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Security BearerAuth
// @Router /devices/{id}/rotate-key [post]
func (h *Handler) RotateKey(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	id, ok := deviceID(c)
	if !ok {
		return
	}

	rotated, err := h.service.RotateKey(c.Request.Context(), middleware.UserID(c), id)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			problem.Write(c, problem.NotFound, "Device not found.")
			return
		}
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to rotate device key.")
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
// @Failure 401 {object} problem.Problem
// @Failure 404 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Security BearerAuth
// @Router /devices/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	id, ok := deviceID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), middleware.UserID(c), id); err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			problem.Write(c, problem.NotFound, "Device not found.")
			return
		}
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to delete device.")
		return
	}

	c.Status(http.StatusNoContent)
}

type devicePath struct {
	ID string `uri:"id" validate:"required,uuid4"`
}

func deviceID(c *gin.Context) (uuid.UUID, bool) {
	params := devicePath{ID: c.Param("id")}
	if !input.Validate(c, "path", &params) {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(params.ID)
	if err != nil {
		problem.Write(c, problem.Validation, "One or more path parameters are invalid.", problem.Violation{In: "path", Name: "id", Code: "format", Detail: "id must be a UUIDv4"})
		return uuid.Nil, false
	}
	return id, true
}

package devicelog

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"go-aiq-backend/internal/platform/input"
	"go-aiq-backend/internal/platform/middleware"
	"go-aiq-backend/internal/platform/problem"
)

// Handler holds dependencies for device log HTTP handlers.
type Handler struct {
	service *Service
}

// NewHandler creates a new device log handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Upload godoc
//
// @Summary Upload device log entries (device)
// @Description Stores up to 50 important log entries (warnings, errors, boot, WiFi, OTA events) from the calling device. Limited to 20 uploads per device per hour.
// @Tags Device Data
// @Accept json
// @Produce json
//
// @Param X-Device-ID header string true "Public device id (dev_...)"
// @Param X-Device-Key header string true "Secret device key (sk_...)"
// @Param request body UploadRequest true "Log batch"
//
// @Success 201 {object} UploadResponse
// @Failure 401 {object} problem.Problem
// @Failure 413 {object} problem.Problem
// @Failure 415 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 429 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Router /device/logs [post]
func (h *Handler) Upload(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	var req UploadRequest
	if !input.BindJSON(c, &req) {
		return
	}
	result, err := h.service.Ingest(c.Request.Context(), middleware.DeviceID(c), req)
	if err != nil {
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to store logs.")
		return
	}
	c.JSON(http.StatusCreated, UploadResponse{Data: *result})
}

// List godoc
//
// @Summary List a device's logs
// @Description Returns the device's log entries, newest first. level is the minimum level (warn returns warn and error). Page with next_before.
// @Tags Devices
// @Produce json
//
// @Param id path string true "Device id (UUID)"
// @Param level query string false "Minimum level" Enums(debug, info, warn, error)
// @Param limit query int false "Page size (1-200, default 50)"
// @Param before query int false "Return entries older than this id (from next_before)"
//
// @Success 200 {object} ListResponse
// @Failure 401 {object} problem.Problem
// @Failure 404 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Security BearerAuth
// @Router /devices/{id}/logs [get]
func (h *Handler) List(c *gin.Context) {
	if !input.RejectUnknownQuery(c, "level", "limit", "before") {
		return
	}
	id, ok := input.PathUUID(c, "id")
	if !ok {
		return
	}
	var query ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		problem.Write(c, problem.Validation, "One or more query parameters are invalid.", problem.Violation{In: "query", Name: "query", Code: "invalid_type", Detail: "limit and before must be integers"})
		return
	}
	if !input.Validate(c, "query", &query) {
		return
	}
	logs, next, err := h.service.List(c.Request.Context(), middleware.UserID(c), id, query)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			problem.Write(c, problem.NotFound, "Device not found.")
			return
		}
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to list logs.")
		return
	}
	c.IndentedJSON(http.StatusOK, ListResponse{Data: logs, NextBefore: next})
}

package firmware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/input"
	"go-aiq-backend/internal/platform/middleware"
	"go-aiq-backend/internal/platform/problem"
)

// Handler holds dependencies for firmware and OTA HTTP handlers.
type Handler struct {
	service *Service
}

// NewHandler creates a new firmware handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Publish godoc
//
// @Summary Publish a firmware release
// @Description Registers an OTA image hosted at an HTTPS URL. Called by the device release pipeline with the FIRMWARE_PUBLISH_TOKEN bearer token. Owners are then offered the update in the app.
// @Tags Firmware
// @Accept json
// @Produce json
//
// @Param Authorization header string true "Bearer <FIRMWARE_PUBLISH_TOKEN>"
// @Param request body PublishReleaseRequest true "Release metadata"
//
// @Success 201 {object} ReleaseResponse
// @Failure 401 {object} problem.Problem
// @Failure 409 {object} problem.Problem
// @Failure 415 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Router /firmware/releases [post]
func (h *Handler) Publish(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	var req PublishReleaseRequest
	if !input.BindJSON(c, &req) {
		return
	}
	release, err := h.service.Publish(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidVersion):
			problem.Write(c, problem.Validation, "The version must be a semantic version.", problem.Violation{In: "body", Name: "version", Code: "format", Detail: "version must look like 1.2.3"})
		case errors.Is(err, ErrImageTooLarge):
			problem.Write(c, problem.Validation, "The image does not fit an OTA slot.", problem.Violation{In: "body", Name: "size", Code: "max", Detail: "size must not exceed the OTA slot size"})
		case errors.Is(err, ErrReleaseExists):
			problem.Write(c, problem.Conflict, "A release with this version already exists.")
		default:
			_ = c.Error(err)
			problem.Write(c, problem.Internal, "Failed to publish release.")
		}
		return
	}
	c.IndentedJSON(http.StatusCreated, ReleaseResponse{Data: *release})
}

// DeviceOta godoc
//
// @Summary Get a device's firmware update state
// @Description Returns the running firmware version, the newest newer release (if any) and the latest update job.
// @Tags Firmware
// @Produce json
//
// @Param id path string true "Device id (UUID)"
//
// @Success 200 {object} DeviceOtaResponse
// @Failure 401 {object} problem.Problem
// @Failure 404 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Security BearerAuth
// @Router /devices/{id}/ota [get]
func (h *Handler) DeviceOta(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	id, ok := input.PathUUID(c, "id")
	if !ok {
		return
	}
	state, err := h.service.DeviceOta(c.Request.Context(), middleware.UserID(c), id)
	if err != nil {
		h.writeError(c, err, "Failed to get the update state.")
		return
	}
	c.IndentedJSON(http.StatusOK, DeviceOtaResponse{Data: *state})
}

// Approve godoc
//
// @Summary Approve a firmware update
// @Description Approves installing a release on the device. The device downloads it on its next check-in (after its next upload, or immediately when the app sends OTA:CHECK over BLE).
// @Tags Firmware
// @Accept json
// @Produce json
//
// @Param id path string true "Device id (UUID)"
// @Param request body ApproveRequest true "Release to install"
//
// @Success 200 {object} JobResponse
// @Failure 401 {object} problem.Problem
// @Failure 404 {object} problem.Problem
// @Failure 409 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Security BearerAuth
// @Router /devices/{id}/ota/approve [post]
func (h *Handler) Approve(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	id, ok := input.PathUUID(c, "id")
	if !ok {
		return
	}
	var req ApproveRequest
	if !input.BindJSON(c, &req) {
		return
	}
	releaseID, _ := uuid.Parse(req.ReleaseID)
	job, err := h.service.Approve(c.Request.Context(), middleware.UserID(c), id, releaseID)
	if err != nil {
		h.writeError(c, err, "Failed to approve the update.")
		return
	}
	c.IndentedJSON(http.StatusOK, JobResponse{Data: *job})
}

// Cancel godoc
//
// @Summary Cancel an approved firmware update
// @Description Withdraws an approval before the device starts downloading.
// @Tags Firmware
// @Produce json
//
// @Param id path string true "Device id (UUID)"
//
// @Success 200 {object} JobResponse
// @Failure 401 {object} problem.Problem
// @Failure 404 {object} problem.Problem
// @Failure 409 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Security BearerAuth
// @Router /devices/{id}/ota/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	id, ok := input.PathUUID(c, "id")
	if !ok {
		return
	}
	job, err := h.service.Cancel(c.Request.Context(), middleware.UserID(c), id)
	if err != nil {
		h.writeError(c, err, "Failed to cancel the update.")
		return
	}
	c.IndentedJSON(http.StatusOK, JobResponse{Data: *job})
}

// DeviceCheck godoc
//
// @Summary Check for an approved update (device)
// @Description Returns the update the owner approved for the calling device, or null.
// @Tags Device Data
// @Produce json
//
// @Param X-Device-ID header string true "Public device id (dev_...)"
// @Param X-Device-Key header string true "Secret device key (sk_...)"
// @Param X-Firmware-Version header string false "Running firmware version"
//
// @Success 200 {object} DeviceJobResponse
// @Failure 401 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Router /device/ota [get]
func (h *Handler) DeviceCheck(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	job, err := h.service.DeviceCheck(c.Request.Context(), middleware.DeviceID(c))
	if err != nil {
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to check for updates.")
		return
	}
	c.JSON(http.StatusOK, DeviceJobResponse{Data: job})
}

// DeviceReport godoc
//
// @Summary Report update progress (device)
// @Description The device reports downloading (with progress), installing, succeeded or failed. 409 means the job was cancelled or already finished and the device must stop.
// @Tags Device Data
// @Accept json
// @Produce json
//
// @Param X-Device-ID header string true "Public device id (dev_...)"
// @Param X-Device-Key header string true "Secret device key (sk_...)"
// @Param request body StatusReportRequest true "Progress report"
//
// @Success 200 {object} JobResponse
// @Failure 401 {object} problem.Problem
// @Failure 404 {object} problem.Problem
// @Failure 409 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Router /device/ota/status [post]
func (h *Handler) DeviceReport(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}
	var req StatusReportRequest
	if !input.BindJSON(c, &req) {
		return
	}
	job, err := h.service.DeviceReport(c.Request.Context(), middleware.DeviceID(c), req)
	if err != nil {
		h.writeError(c, err, "Failed to record the update status.")
		return
	}
	c.JSON(http.StatusOK, JobResponse{Data: *job})
}

func (h *Handler) writeError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, ErrDeviceNotFound):
		problem.Write(c, problem.NotFound, "Device not found.")
	case errors.Is(err, ErrReleaseNotFound):
		problem.Write(c, problem.NotFound, "Release not found.")
	case errors.Is(err, ErrJobNotFound):
		problem.Write(c, problem.NotFound, "Update job not found.")
	case errors.Is(err, ErrFirmwareUnknown):
		problem.Write(c, problem.Validation, "The device has not reported a firmware version yet; it must run OTA-capable firmware.", problem.Violation{In: "path", Name: "id", Code: "unsupported", Detail: err.Error()})
	case errors.Is(err, ErrNotNewer):
		problem.Write(c, problem.Validation, "The release is not newer than the device firmware.", problem.Violation{In: "body", Name: "release_id", Code: "not_newer", Detail: err.Error()})
	case errors.Is(err, ErrUpdateInProgress):
		problem.Write(c, problem.Conflict, "An update is already downloading or installing.")
	case errors.Is(err, ErrNothingToCancel):
		problem.Write(c, problem.Conflict, "There is no approved update to cancel.")
	case errors.Is(err, ErrInvalidTransition):
		problem.Write(c, problem.Conflict, "The update job no longer accepts this status.")
	default:
		_ = c.Error(err)
		problem.Write(c, problem.Internal, fallback)
	}
}

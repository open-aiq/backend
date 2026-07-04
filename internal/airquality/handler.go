package airquality

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler holds dependencies for air quality HTTP handlers.
type Handler struct {
	service *Service
}

// NewHandler creates a new air quality handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetCurrent godoc
//
// @Summary Get current air quality
// @Description Returns sensor metrics averaged over the last hour, the device status (offline when nothing was received for 20 minutes), the last_seen timestamp, and the latest known location. 404 only when the device has never reported.
// @Tags Air Quality
// @Produce json
//
// @Success 200 {object} CurrentResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /air-quality/current [get]
func (h *Handler) GetCurrent(c *gin.Context) {
	h.respondCurrent(c, nil)
}

// GetDeviceCurrent godoc
//
// @Summary Get current air quality for one device
// @Description Same as /air-quality/current but restricted to a single device
// @Tags Air Quality
// @Produce json
//
// @Param id path string true "Device id (UUID)"
//
// @Success 200 {object} CurrentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /devices/{id}/current [get]
func (h *Handler) GetDeviceCurrent(c *gin.Context) {
	id, ok := deviceIDParam(c)
	if !ok {
		return
	}
	h.respondCurrent(c, &id)
}

// respondCurrent renders the current aggregate, optionally scoped to a device.
func (h *Handler) respondCurrent(c *gin.Context, deviceID *uuid.UUID) {
	current, err := h.service.GetCurrent(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, ErrNoData) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "No readings received yet"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get current data"})
		return
	}

	c.IndentedJSON(http.StatusOK, CurrentResponse{Data: *current})
}

// deviceIDParam parses the :id path param as a UUID, responding with 400 on
// failure.
func deviceIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid device id",
			Details: "id must be a UUID",
		})
		return uuid.Nil, false
	}
	return id, true
}

// GetHistorical godoc
//
// @Summary Get historical air quality data
// @Description Returns historical air quality data aggregated by timeline
// @Tags Air Quality
// @Produce json
//
// @Param timeline query string true "Timeline" Enums(daily, weekly, monthly, yearly)
//
// @Success 200 {object} HistoricalResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /air-quality/historical [get]
func (h *Handler) GetHistorical(c *gin.Context) {
	h.respondHistorical(c, nil)
}

// GetDeviceHistorical godoc
//
// @Summary Get historical air quality data for one device
// @Description Same as /air-quality/historical but restricted to a single device
// @Tags Air Quality
// @Produce json
//
// @Param id path string true "Device id (UUID)"
// @Param timeline query string true "Timeline" Enums(daily, weekly, monthly, yearly)
//
// @Success 200 {object} HistoricalResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /devices/{id}/historical [get]
func (h *Handler) GetDeviceHistorical(c *gin.Context) {
	id, ok := deviceIDParam(c)
	if !ok {
		return
	}
	h.respondHistorical(c, &id)
}

// respondHistorical renders the bucketed timeline, optionally scoped to a device.
func (h *Handler) respondHistorical(c *gin.Context, deviceID *uuid.UUID) {
	var query HistoricalQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Details: "timeline is required and must be one of: daily, weekly, monthly, yearly",
		})
		return
	}

	data, err := h.service.GetHistorical(c.Request.Context(), query.Timeline, deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get historical data"})
		return
	}

	c.IndentedJSON(http.StatusOK, HistoricalResponse{Timeline: query.Timeline, Data: data})
}

// GetCustomRange godoc
//
// @Summary Get air quality data for a custom date range
// @Description Returns air quality data points between start_date and end_date
// @Tags Air Quality
// @Produce json
//
// @Param start_date query string true "Start date (YYYY-MM-DD)"
// @Param end_date query string true "End date (YYYY-MM-DD)"
//
// @Success 200 {object} CustomRangeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /air-quality/custom [get]
func (h *Handler) GetCustomRange(c *gin.Context) {
	var query CustomQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Details: "start_date and end_date are required (YYYY-MM-DD)",
		})
		return
	}

	start, err := time.Parse("2006-01-02", query.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid start_date format, use YYYY-MM-DD"})
		return
	}

	end, err := time.Parse("2006-01-02", query.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid end_date format, use YYYY-MM-DD"})
		return
	}

	if end.Before(start) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "end_date must be after start_date"})
		return
	}

	data, err := h.service.GetCustomRange(c.Request.Context(), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get custom range data"})
		return
	}

	c.IndentedJSON(http.StatusOK, CustomRangeResponse{Data: data})
}

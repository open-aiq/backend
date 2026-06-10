package airquality

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
// @Description Returns the latest air quality reading
// @Tags Air Quality
// @Produce json
//
// @Success 200 {object} CurrentResponse
// @Failure 500 {object} ErrorResponse
//
// @Router /air-quality/current [get]
func (h *Handler) GetCurrent(c *gin.Context) {
	current, err := h.service.GetCurrent(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current data"})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"status":  "success",
		"current": current,
	})
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
	var query HistoricalQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": "timeline is required and must be one of: daily, weekly, monthly, yearly",
		})
		return
	}

	historical, err := h.service.GetHistorical(c.Request.Context(), query.Timeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get historical data"})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"status":     "success",
		"timeline":   query.Timeline,
		"historical": historical,
	})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": "start_date and end_date are required (YYYY-MM-DD)",
		})
		return
	}

	start, err := time.Parse("2006-01-02", query.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format, use YYYY-MM-DD"})
		return
	}

	end, err := time.Parse("2006-01-02", query.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format, use YYYY-MM-DD"})
		return
	}

	if end.Before(start) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_date must be after start_date"})
		return
	}

	data, err := h.service.GetCustomRange(c.Request.Context(), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get custom range data"})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"status":     "success",
		"start_date": query.StartDate,
		"end_date":   query.EndDate,
		"data":       data,
	})
}

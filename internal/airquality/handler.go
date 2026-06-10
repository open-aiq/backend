package airquality

import (
	"net/http"

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

// GetAirQuality godoc
//
// @Summary Get air quality data
// @Description Returns current and historical air quality information
// @Tags Air Quality
// @Accept json
// @Produce json
//
// @Param timeline query string false "Timeline (daily, weekly, monthly, yearly, all)"
//
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
//
// @Router /air-quality [get]
func (h *Handler) GetAirQuality(c *gin.Context) {
	var filters FilterQuery

	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing or invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	current, err := h.service.GetCurrent(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current data"})
		return
	}

	historical, err := h.service.GetHistorical(c.Request.Context(), filters.Timeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get historical data"})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"status":     "success",
		"current":    current,
		"historical": historical,
	})
}

package airquality

import "github.com/gin-gonic/gin"

// RegisterRoutes registers air quality routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/air-quality", h.GetAirQuality)
}

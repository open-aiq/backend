package airquality

import "github.com/gin-gonic/gin"

// RegisterRoutes registers air quality routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	aq := rg.Group("/air-quality")
	{
		aq.GET("/current", h.GetCurrent)
		aq.GET("/historical", h.GetHistorical)
		aq.GET("/custom", h.GetCustomRange)
	}
}

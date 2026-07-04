package device

import "github.com/gin-gonic/gin"

// RegisterRoutes registers device routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices")
	{
		devices.POST("", h.Create)
		devices.GET("", h.List)
		devices.PATCH("/:id", h.Update)
		devices.POST("/:id/rotate-key", h.RotateKey)
		devices.DELETE("/:id", h.Delete)
	}
}

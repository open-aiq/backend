package device

import (
	"github.com/gin-gonic/gin"
	"go-aiq-backend/internal/platform/middleware"
)

// RegisterRoutes registers device routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices")
	{
		// Registering a device and rotating its key hand out a secret that only
		// the mobile app can deliver to the monitor over BLE.
		devices.POST("", middleware.RequireNativeClient(), h.Create)
		devices.GET("", h.List)
		devices.PATCH("/:id", h.Update)
		devices.POST("/:id/rotate-key", middleware.RequireNativeClient(), h.RotateKey)
		devices.DELETE("/:id", h.Delete)
	}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	public := rg.Group("/public/devices")
	public.GET("", h.ListPublic)
	public.GET("/:id", h.GetPublic)
}

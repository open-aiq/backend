package firmware

import (
	"github.com/gin-gonic/gin"

	"go-aiq-backend/internal/platform/middleware"
)

// RegisterRoutes registers owner (Clerk-authenticated) OTA routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices/:id/ota")
	devices.GET("", h.DeviceOta)
	devices.POST("/approve", h.Approve)
	devices.POST("/cancel", h.Cancel)
}

// RegisterPublishRoutes registers the release publishing route, authorized by a shared token.
func (h *Handler) RegisterPublishRoutes(rg *gin.RouterGroup, publishToken string) {
	rg.POST("/firmware/releases", middleware.StaticBearer(publishToken), h.Publish)
}

// RegisterDeviceRoutes registers routes called by physical devices.
func (h *Handler) RegisterDeviceRoutes(rg *gin.RouterGroup, auth middleware.DeviceAuthenticator) {
	device := rg.Group("/device/ota", middleware.DeviceAuth(auth))
	device.GET("", h.DeviceCheck)
	device.POST("/status", h.DeviceReport)
}

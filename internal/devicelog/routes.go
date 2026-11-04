package devicelog

import (
	"github.com/gin-gonic/gin"

	"go-aiq-backend/internal/platform/middleware"
)

// RegisterRoutes registers owner (Clerk-authenticated) log routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/devices/:id/logs", h.List)
}

// RegisterDeviceRoutes registers the device log upload route.
func (h *Handler) RegisterDeviceRoutes(rg *gin.RouterGroup, auth middleware.DeviceAuthenticator) {
	rg.POST("/device/logs", middleware.DeviceAuth(auth), h.Upload)
}

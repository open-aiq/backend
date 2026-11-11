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
	// The limiter is in memory, so the budget is per server instance and resets
	// on restart. It runs after DeviceAuth because it is keyed by the
	// authenticated device; requests with bad credentials are not limited here.
	limit := middleware.RateLimit(middleware.NewRateLimiter(UploadsPerWindow, RateWindow), middleware.DeviceKey)
	rg.POST("/device/logs", middleware.DeviceAuth(auth), limit, h.Upload)
}

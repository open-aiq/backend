package devicereading

import (
	"github.com/gin-gonic/gin"

	"go-aiq-backend/internal/platform/middleware"
)

// RegisterRoutes registers device reading routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	data := rg.Group("/data")
	data.Use(middleware.DeviceAuth(h.auth))
	{
		data.POST("", h.Upload)
	}
}

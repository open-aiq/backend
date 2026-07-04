package devicereading

import "github.com/gin-gonic/gin"

// RegisterRoutes registers device reading routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	data := rg.Group("/data")
	data.Use(h.deviceAuth)
	{
		data.POST("", h.Upload)
	}
}

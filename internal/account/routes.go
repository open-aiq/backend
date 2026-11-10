package account

import "github.com/gin-gonic/gin"

// RegisterRoutes registers account routes on the given (authenticated) router
// group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.DELETE("/account", h.Delete)
}

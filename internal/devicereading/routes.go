package devicereading

import (
	"time"

	"github.com/gin-gonic/gin"

	"go-aiq-backend/internal/platform/middleware"
)

// Devices upload once every 10 minutes. The limit is generous so retries and
// reboots never trip it, while a runaway device can't flood the table.
const (
	uploadsPerWindow = 30
	uploadWindow     = time.Minute
)

// RegisterRoutes registers device reading routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	data := rg.Group("/data")
	// The limiter is in memory, so the budget is per server instance and resets
	// on restart. It runs after DeviceAuth because it is keyed by the
	// authenticated device (keying on the unauthenticated X-Device-ID header
	// would let anyone lock a device out); requests with bad credentials are
	// therefore not limited here and still reach the credential lookup.
	data.Use(
		middleware.DeviceAuth(h.auth),
		middleware.RateLimit(middleware.NewRateLimiter(uploadsPerWindow, uploadWindow), middleware.DeviceKey),
	)
	{
		data.POST("", h.Upload)
	}
}

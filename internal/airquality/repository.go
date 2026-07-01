package airquality

import (
	"context"
	"time"
)

// Repository defines the data access interface for air quality data.
type Repository interface {
	GetCurrent(ctx context.Context) (*AirQuality, error)
	GetHistorical(ctx context.Context, timeline string) ([]DataPoint, error)
	GetCustomRange(ctx context.Context, start, end time.Time) ([]DataPoint, error)
}

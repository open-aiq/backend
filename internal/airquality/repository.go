package airquality

import "context"

// Repository defines the data access interface for air quality data.
type Repository interface {
	GetCurrent(ctx context.Context) (*AirQuality, error)
	GetHistorical(ctx context.Context, timeline string) (*HistoricalData, error)
}

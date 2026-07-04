package airquality

import "time"

// AirQuality represents aggregated sensor metrics: particulate matter,
// calculated AQI, and temperature data.
type AirQuality struct {
	PM10        float64 `json:"pm1_0"`
	PM25        float64 `json:"pm2_5"`
	PM100       float64 `json:"pm10_0"`
	AQI         int     `json:"aqi"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	HeatIndex   float64 `json:"heat_index"`
}

// Location is the latest known location fix of the sensor.
type Location struct {
	Lat       float64   `json:"lat" example:"24.8607"`
	Lon       float64   `json:"lon" example:"67.0011"`
	Provider  string    `json:"provider" example:"mobile"`
	Timestamp time.Time `json:"timestamp"`
}

// Device status values reported by the current endpoint.
const (
	StatusOnline  = "online"
	StatusOffline = "offline"
)

// CurrentAirQuality is the last-hour aggregate plus the device status and the
// latest known location. Metrics are zero (with sample_count 0) when the
// device has been offline for longer than the aggregation window.
type CurrentAirQuality struct {
	Status   string    `json:"status" example:"online"`
	LastSeen time.Time `json:"last_seen"`
	AirQuality
	SampleCount int       `json:"sample_count" example:"6"`
	Location    *Location `json:"location,omitempty"`
}

// DataPoint represents a single data point in a time series
type DataPoint struct {
	Timestamp time.Time  `json:"timestamp"`
	Label     string     `json:"label"`
	Metrics   AirQuality `json:"metrics"`
}

// HistoricalQuery holds query parameters for the historical endpoint.
type HistoricalQuery struct {
	Timeline string `form:"timeline" binding:"required,oneof=daily weekly monthly yearly"`
}

// CustomQuery holds query parameters for the custom date range endpoint.
type CustomQuery struct {
	StartDate string `form:"start_date" binding:"required"`
	EndDate   string `form:"end_date" binding:"required"`
}

// Response types for swagger documentation

type CurrentResponse struct {
	Data CurrentAirQuality `json:"data"`
}

type HistoricalResponse struct {
	Timeline string      `json:"timeline" example:"daily"`
	Data     []DataPoint `json:"data"`
}

type CustomRangeResponse struct {
	Data []DataPoint `json:"data"`
}

type ErrorResponse struct {
	Error   string `json:"error" example:"Missing or invalid query parameters"`
	Details string `json:"details,omitempty" example:"timeline is required"`
}

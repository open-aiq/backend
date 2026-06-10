package airquality

import "time"

// AirQuality represents current particulate matter and calculated AQI
type AirQuality struct {
	PM10  float64 `json:"pm1_0"`
	PM25  float64 `json:"pm2_5"`
	PM100 float64 `json:"pm10_0"`
	AQI   int     `json:"aqi"`
}

// HistoricalData wraps different time-series aggregations
type HistoricalData struct {
	DailyHourly   []DataPoint `json:"daily_hourly,omitempty"`
	WeeklyDaily   []DataPoint `json:"weekly_daily,omitempty"`
	MonthlyWeekly []DataPoint `json:"monthly_weekly,omitempty"`
	YearlyMonthly []DataPoint `json:"yearly_monthly,omitempty"`
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
	Status  string     `json:"status" example:"success"`
	Current AirQuality `json:"current"`
}

type HistoricalResponse struct {
	Status     string         `json:"status" example:"success"`
	Timeline   string         `json:"timeline" example:"daily"`
	Historical HistoricalData `json:"historical"`
}

type CustomRangeResponse struct {
	Status    string      `json:"status" example:"success"`
	StartDate string      `json:"start_date" example:"2026-01-01"`
	EndDate   string      `json:"end_date" example:"2026-06-10"`
	Data      []DataPoint `json:"data"`
}

type ErrorResponse struct {
	Error   string `json:"error" example:"Missing or invalid query parameters"`
	Details string `json:"details,omitempty" example:"timeline is required"`
}

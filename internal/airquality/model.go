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
	Label     string     `json:"label"` // e.g., "14:00", "Monday", "Week 3", "January"
	Metrics   AirQuality `json:"metrics"`
}

// FilterQuery handles the incoming URL query parameters
type FilterQuery struct {
	Timeline string `form:"timeline"` // e.g., "all", "daily", "weekly"
}

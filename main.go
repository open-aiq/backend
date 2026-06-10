//go:generate swag init -g main.go

package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "go-aiq-backend/docs"
)

// --- Models ---

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

type DataPoint struct {
	Timestamp time.Time  `json:"timestamp"`
	Label     string     `json:"label"` // e.g., "14:00", "Monday", "Week 3", "January"
	Metrics   AirQuality `json:"metrics"`
}

// FilterQuery handles the incoming URL query parameters
type FilterQuery struct {
	Location string `form:"location" binding:"required"` // Custom filter (Required)
	SensorID string `form:"sensor_id"`                   // Custom filter (Optional)
	Timeline string `form:"timeline"`                    // e.g., "all", "daily", "weekly"
}

// @title Air Quality API
// @version 1.0
// @description Air quality monitoring API
// @host localhost:8080
// @BasePath /api/v1
func main() {
	// Initialize Gin engine with default logger and recovery middleware
	r := gin.Default()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API Route Group
	api := r.Group("/api/v1")
	{
		api.GET("/air-quality", handleAirQuality)
	}

	// Start server on port 8080
	r.Run(":8080")
}

// handleAirQuality godoc
//
// @Summary Get air quality data
// @Description Returns current and historical air quality information
// @Tags Air Quality
// @Accept json
// @Produce json
//
// @Param location query string true "Location"
// @Param sensor_id query string false "Sensor ID"
// @Param timeline query string false "Timeline (daily, weekly, monthly, yearly, all)"
//
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
//
// @Router /air-quality [get]
func handleAirQuality(c *gin.Context) {
	var filters FilterQuery

	// Bind query parameters to our FilterQuery struct.
	// If 'location' is missing, it will automatically return a 400 Bad Request.
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing or invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Generate baseline current data
	currentAQI := AirQuality{
		PM10:  12.5,
		PM25:  35.2,
		PM100: 45.0,
		AQI:   98, // Moderate
	}

	// Build historical payload based on requested timeline filter
	historical := HistoricalData{}
	now := time.Now()

	if filters.Timeline == "daily" || filters.Timeline == "all" || filters.Timeline == "" {
		// Mock 24 hours of data
		for i := range 24 {
			historical.DailyHourly = append(historical.DailyHourly, DataPoint{
				Timestamp: now.Add(time.Duration(-i) * time.Hour),
				Label:     now.Add(time.Duration(-i) * time.Hour).Format("15:00"),
				Metrics:   AirQuality{PM10: 11.0, PM25: 30.5, PM100: 40.0, AQI: 85},
			})
		}
	}

	if filters.Timeline == "weekly" || filters.Timeline == "all" || filters.Timeline == "" {
		// Mock 7 days of data
		days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		for i := range 7 {
			historical.WeeklyDaily = append(historical.WeeklyDaily, DataPoint{
				Timestamp: now.AddDate(0, 0, -i),
				Label:     days[int(now.AddDate(0, 0, -i).Weekday())],
				Metrics:   AirQuality{PM10: 14.2, PM25: 42.1, PM100: 55.0, AQI: 110},
			})
		}
	}

	if filters.Timeline == "monthly" || filters.Timeline == "all" || filters.Timeline == "" {
		// Mock 4 weeks of data
		for i := 1; i <= 4; i++ {
			historical.MonthlyWeekly = append(historical.MonthlyWeekly, DataPoint{
				Timestamp: now.AddDate(0, 0, -i*7),
				Label:     "Week " + string(rune(53-i)), // Arbitrary week identifier
				Metrics:   AirQuality{PM10: 9.8, PM25: 22.4, PM100: 31.0, AQI: 72},
			})
		}
	}

	if filters.Timeline == "yearly" || filters.Timeline == "all" || filters.Timeline == "" {
		// Mock 12 months of data
		for i := range 12 {
			t := now.AddDate(0, -i, 0)
			historical.YearlyMonthly = append(historical.YearlyMonthly, DataPoint{
				Timestamp: t,
				Label:     t.Format("January"),
				Metrics:   AirQuality{PM10: 15.0, PM25: 55.0, PM100: 68.0, AQI: 145},
			})
		}
	}

	// Construct and send the final response contextually filtered
	c.IndentedJSON(http.StatusOK, gin.H{
		"status":     "success",
		"location":   filters.Location,
		"sensor_id":  filters.SensorID,
		"current":    currentAQI,
		"historical": historical,
	})
}

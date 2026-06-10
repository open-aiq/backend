package airquality

import (
	"context"
	"time"
)

// Service handles air quality business logic.
type Service struct {
	repo Repository
}

// NewService creates a new air quality service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetCurrent returns the current air quality data.
func (s *Service) GetCurrent(ctx context.Context) (*AirQuality, error) {
	return s.repo.GetCurrent(ctx)
}

// GetHistorical returns historical air quality data based on the timeline filter.
func (s *Service) GetHistorical(ctx context.Context, timeline string) (*HistoricalData, error) {
	return s.repo.GetHistorical(ctx, timeline)
}

// MockRepository is a temporary in-memory implementation of Repository.
type MockRepository struct{}

func NewMockRepository() *MockRepository {
	return &MockRepository{}
}

func (r *MockRepository) GetCurrent(_ context.Context) (*AirQuality, error) {
	return &AirQuality{
		PM10:  12.5,
		PM25:  35.2,
		PM100: 45.0,
		AQI:   98,
	}, nil
}

func (r *MockRepository) GetHistorical(_ context.Context, timeline string) (*HistoricalData, error) {
	historical := &HistoricalData{}
	now := time.Now()

	if timeline == "daily" || timeline == "all" || timeline == "" {
		for i := range 24 {
			historical.DailyHourly = append(historical.DailyHourly, DataPoint{
				Timestamp: now.Add(time.Duration(-i) * time.Hour),
				Label:     now.Add(time.Duration(-i) * time.Hour).Format("15:00"),
				Metrics:   AirQuality{PM10: 11.0, PM25: 30.5, PM100: 40.0, AQI: 85},
			})
		}
	}

	if timeline == "weekly" || timeline == "all" || timeline == "" {
		days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		for i := range 7 {
			historical.WeeklyDaily = append(historical.WeeklyDaily, DataPoint{
				Timestamp: now.AddDate(0, 0, -i),
				Label:     days[int(now.AddDate(0, 0, -i).Weekday())],
				Metrics:   AirQuality{PM10: 14.2, PM25: 42.1, PM100: 55.0, AQI: 110},
			})
		}
	}

	if timeline == "monthly" || timeline == "all" || timeline == "" {
		for i := 1; i <= 4; i++ {
			historical.MonthlyWeekly = append(historical.MonthlyWeekly, DataPoint{
				Timestamp: now.AddDate(0, 0, -i*7),
				Label:     "Week " + string(rune(53-i)),
				Metrics:   AirQuality{PM10: 9.8, PM25: 22.4, PM100: 31.0, AQI: 72},
			})
		}
	}

	if timeline == "yearly" || timeline == "all" || timeline == "" {
		for i := range 12 {
			t := now.AddDate(0, -i, 0)
			historical.YearlyMonthly = append(historical.YearlyMonthly, DataPoint{
				Timestamp: t,
				Label:     t.Format("January"),
				Metrics:   AirQuality{PM10: 15.0, PM25: 55.0, PM100: 68.0, AQI: 145},
			})
		}
	}

	return historical, nil
}

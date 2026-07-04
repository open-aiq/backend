package airquality

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrNoData is returned when no readings exist at all.
var ErrNoData = errors.New("no air quality data")

const (
	// currentWindow is how far back the "current" aggregate looks.
	currentWindow = time.Hour
	// offlineThreshold marks the device offline after two missed samples
	// (devices report every 10 minutes).
	offlineThreshold = 20 * time.Minute
)

// Service handles air quality business logic.
type Service struct {
	repo Repository
}

// NewService creates a new air quality service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetCurrent returns metrics averaged over the last hour, the device status
// (offline once the newest reading is older than offlineThreshold), and the
// latest known location. A non-nil deviceID restricts everything to that
// device; nil aggregates across all devices. It returns ErrNoData only when no
// readings exist at all (device never reported).
func (s *Service) GetCurrent(ctx context.Context, deviceID *uuid.UUID) (*CurrentAirQuality, error) {
	lastSeen, err := s.repo.LastSeen(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get current: %w", err)
	}
	if lastSeen == nil {
		return nil, ErrNoData
	}

	status := StatusOnline
	if time.Since(*lastSeen) > offlineThreshold {
		status = StatusOffline
	}

	metrics, count, err := s.repo.AverageSince(ctx, time.Now().Add(-currentWindow), deviceID)
	if err != nil {
		return nil, fmt.Errorf("get current: %w", err)
	}
	if metrics == nil {
		// Offline for longer than the window: nothing to average.
		metrics = &AirQuality{}
	}

	location, err := s.repo.LatestLocation(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get current: %w", err)
	}

	return &CurrentAirQuality{
		Status:      status,
		LastSeen:    *lastSeen,
		AirQuality:  *metrics,
		SampleCount: count,
		Location:    location,
	}, nil
}

// timelineSpec defines how a timeline value maps to a query window, a
// date_trunc bucket, and a point label.
type timelineSpec struct {
	start  time.Time
	bucket string
	label  func(time.Time) string
}

// spec returns the timelineSpec for a validated timeline value.
func spec(timeline string, now time.Time) (timelineSpec, error) {
	switch timeline {
	case "daily": // last 24 hours by hour
		return timelineSpec{
			start:  now.Add(-24 * time.Hour),
			bucket: "hour",
			label:  func(t time.Time) string { return t.Format("15:04") },
		}, nil
	case "weekly": // last 7 days by day
		return timelineSpec{
			start:  now.AddDate(0, 0, -7),
			bucket: "day",
			label:  func(t time.Time) string { return t.Weekday().String() },
		}, nil
	case "monthly": // last 30 days by day
		return timelineSpec{
			start:  now.AddDate(0, 0, -30),
			bucket: "day",
			label:  func(t time.Time) string { return t.Format("2006-01-02") },
		}, nil
	case "yearly": // last 12 months by month
		return timelineSpec{
			start:  now.AddDate(-1, 0, 0),
			bucket: "month",
			label:  func(t time.Time) string { return t.Format("January") },
		}, nil
	default:
		return timelineSpec{}, fmt.Errorf("invalid timeline %q", timeline)
	}
}

// GetHistorical returns bucketed averages for the given timeline: daily (24h by
// hour), weekly (7 days by day), monthly (30 days by day), yearly (12 months by
// month). A non-nil deviceID restricts the data to that device. Only buckets
// containing data are returned.
func (s *Service) GetHistorical(ctx context.Context, timeline string, deviceID *uuid.UUID) ([]DataPoint, error) {
	now := time.Now()
	ts, err := spec(timeline, now)
	if err != nil {
		return nil, err
	}

	buckets, err := s.repo.BucketedAverages(ctx, ts.start, now, ts.bucket, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get historical: %w", err)
	}

	return toDataPoints(buckets, ts.label), nil
}

// GetCustomRange returns daily averages between start and end (inclusive),
// across all devices.
func (s *Service) GetCustomRange(ctx context.Context, start, end time.Time) ([]DataPoint, error) {
	// end is a date; extend it by a day so the whole end date is included.
	buckets, err := s.repo.BucketedAverages(ctx, start, end.AddDate(0, 0, 1), "day", nil)
	if err != nil {
		return nil, fmt.Errorf("get custom range: %w", err)
	}

	return toDataPoints(buckets, func(t time.Time) string { return t.Format("2006-01-02") }), nil
}

// toDataPoints maps aggregated buckets to labeled API data points.
func toDataPoints(buckets []BucketPoint, label func(time.Time) string) []DataPoint {
	points := make([]DataPoint, 0, len(buckets))
	for _, b := range buckets {
		points = append(points, DataPoint{
			Timestamp: b.Bucket,
			Label:     label(b.Bucket),
			Metrics:   b.Metrics,
		})
	}
	return points
}

package airquality

import (
	"context"
	"fmt"
	"math"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/ent"
	entreading "go-aiq-backend/internal/platform/ent/devicereading"
)

// BucketPoint is one aggregated time bucket returned by BucketedAverages.
type BucketPoint struct {
	Bucket  time.Time
	Metrics AirQuality
}

// Repository defines the data access interface for aggregated air quality
// data. On every method, a non-nil deviceID restricts the readings to that
// device (by internal id); nil aggregates across all devices.
type Repository interface {
	// AverageSince averages all metrics over readings created at or after
	// since, returning the sample count (0 when there is no data).
	AverageSince(ctx context.Context, since time.Time, deviceID *uuid.UUID) (*AirQuality, int, error)
	// BucketedAverages groups readings in [start, end) into date_trunc buckets
	// ("hour", "day" or "month") and averages each metric. Only buckets that
	// contain data are returned, ordered chronologically.
	BucketedAverages(ctx context.Context, start, end time.Time, bucket string, deviceID *uuid.UUID) ([]BucketPoint, error)
	// LatestLocation returns the location of the most recent reading that has
	// a location fix, or nil if none exists.
	LatestLocation(ctx context.Context, deviceID *uuid.UUID) (*Location, error)
	// LastSeen returns the timestamp of the most recent reading, or nil if no
	// readings exist at all.
	LastSeen(ctx context.Context, deviceID *uuid.UUID) (*time.Time, error)
}

// entRepository is an Ent-backed implementation of Repository over device readings.
type entRepository struct {
	client *ent.Client
}

// NewEntRepository creates an air quality repository backed by the Ent client.
func NewEntRepository(client *ent.Client) Repository {
	return &entRepository{client: client}
}

// avgColumns selects avg() of every metric column under its own column name,
// so rows scan into avgRow via its `sql` tags.
func avgColumns() []string {
	return []string{
		entsql.As("avg(pm1_0)", "pm1_0"),
		entsql.As("avg(pm2_5)", "pm2_5"),
		entsql.As("avg(pm10_0)", "pm10_0"),
		entsql.As("avg(aqi)", "aqi"),
		entsql.As("avg(temperature)", "temperature"),
		entsql.As("avg(humidity)", "humidity"),
		entsql.As("avg(heat_index)", "heat_index"),
	}
}

// avgRow receives avg() aggregates. Metric pointers because avg() over zero
// rows is NULL. SampleCount and Bucket are only populated by the queries that
// select those columns (the scanner maps selected columns to fields by tag).
type avgRow struct {
	PM10        *float64  `sql:"pm1_0"`
	PM25        *float64  `sql:"pm2_5"`
	PM100       *float64  `sql:"pm10_0"`
	AQI         *float64  `sql:"aqi"`
	Temperature *float64  `sql:"temperature"`
	Humidity    *float64  `sql:"humidity"`
	HeatIndex   *float64  `sql:"heat_index"`
	SampleCount int       `sql:"sample_count"`
	Bucket      time.Time `sql:"bucket"`
}

// metrics converts a scanned row to AirQuality, rounding the averaged AQI.
func (r avgRow) metrics() AirQuality {
	deref := func(f *float64) float64 {
		if f == nil {
			return 0
		}
		return *f
	}
	return AirQuality{
		PM10:        deref(r.PM10),
		PM25:        deref(r.PM25),
		PM100:       deref(r.PM100),
		AQI:         int(math.Round(deref(r.AQI))),
		Temperature: deref(r.Temperature),
		Humidity:    deref(r.Humidity),
		HeatIndex:   deref(r.HeatIndex),
	}
}

// scopedQuery starts a reading query, optionally restricted to one device.
func (r *entRepository) scopedQuery(deviceID *uuid.UUID) *ent.DeviceReadingQuery {
	q := r.client.DeviceReading.Query()
	if deviceID != nil {
		q = q.Where(entreading.DeviceIDEQ(*deviceID))
	}
	return q
}

func (r *entRepository) AverageSince(ctx context.Context, since time.Time, deviceID *uuid.UUID) (*AirQuality, int, error) {
	var rows []avgRow

	err := r.scopedQuery(deviceID).
		Where(entreading.CreatedAtGTE(since)).
		Modify(func(s *entsql.Selector) {
			s.Select(append(avgColumns(), entsql.As("count(*)", "sample_count"))...)
		}).
		Scan(ctx, &rows)
	if err != nil {
		return nil, 0, fmt.Errorf("average since %s: %w", since, err)
	}
	if len(rows) == 0 || rows[0].SampleCount == 0 {
		return nil, 0, nil
	}

	m := rows[0].metrics()
	return &m, rows[0].SampleCount, nil
}

func (r *entRepository) BucketedAverages(ctx context.Context, start, end time.Time, bucket string, deviceID *uuid.UUID) ([]BucketPoint, error) {
	// bucket is one of the fixed date_trunc precisions used by the service,
	// never user input.
	switch bucket {
	case "hour", "day", "month":
	default:
		return nil, fmt.Errorf("invalid bucket %q", bucket)
	}

	var rows []avgRow

	err := r.scopedQuery(deviceID).
		Where(
			entreading.CreatedAtGTE(start),
			entreading.CreatedAtLT(end),
		).
		Modify(func(s *entsql.Selector) {
			trunc := fmt.Sprintf("date_trunc('%s', created_at)", bucket)
			s.Select(append(avgColumns(), entsql.As(trunc, "bucket"))...).
				GroupBy("bucket").
				OrderBy("bucket")
		}).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("bucketed averages by %s: %w", bucket, err)
	}

	points := make([]BucketPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, BucketPoint{Bucket: row.Bucket, Metrics: row.metrics()})
	}
	return points, nil
}

func (r *entRepository) LastSeen(ctx context.Context, deviceID *uuid.UUID) (*time.Time, error) {
	reading, err := r.scopedQuery(deviceID).
		Order(entreading.ByCreatedAt(entsql.OrderDesc())).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("last seen: %w", err)
	}
	return &reading.CreatedAt, nil
}

func (r *entRepository) LatestLocation(ctx context.Context, deviceID *uuid.UUID) (*Location, error) {
	reading, err := r.scopedQuery(deviceID).
		Where(entreading.LatNotNil(), entreading.LonNotNil()).
		Order(entreading.ByCreatedAt(entsql.OrderDesc())).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("latest location: %w", err)
	}

	return &Location{
		Lat:       *reading.Lat,
		Lon:       *reading.Lon,
		Provider:  reading.LocationProvider,
		Timestamp: reading.CreatedAt,
	}, nil
}

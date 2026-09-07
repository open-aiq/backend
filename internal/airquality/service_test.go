package airquality

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mapRepositoryStub struct {
	rows []PublicMapReading
}

func (r mapRepositoryStub) AverageSince(context.Context, time.Time, *uuid.UUID, Scope) (*AirQuality, int, error) {
	return nil, 0, nil
}
func (r mapRepositoryStub) BucketedAverages(context.Context, time.Time, time.Time, string, *uuid.UUID, Scope) ([]BucketPoint, error) {
	return nil, nil
}
func (r mapRepositoryStub) LatestLocation(context.Context, *uuid.UUID, Scope) (*Location, error) {
	return nil, nil
}
func (r mapRepositoryStub) LastSeen(context.Context, *uuid.UUID, Scope) (*time.Time, error) {
	return nil, nil
}
func (r mapRepositoryStub) DeviceAccessible(context.Context, uuid.UUID, Scope) (bool, error) {
	return true, nil
}
func (r mapRepositoryStub) LatestPublicMapReadings(context.Context) ([]PublicMapReading, error) {
	return r.rows, nil
}

func TestGetPublicMapDevicesAssignsStatusAndFields(t *testing.T) {
	now := time.Now()
	onlineID := uuid.New()
	offlineID := uuid.New()
	service := NewService(mapRepositoryStub{rows: []PublicMapReading{
		{ID: onlineID, Name: "Current", AQI: 42, PM25: 8.5, Lat: 24.8, Lon: 67.0, MeasuredAt: now.Add(-5 * time.Minute)},
		{ID: offlineID, Name: "Stale", AQI: 130, IsOutdoor: true, MeasuredAt: now.Add(-21 * time.Minute)},
	}})

	devices, err := service.GetPublicMapDevices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 {
		t.Fatalf("got %d devices, want 2", len(devices))
	}
	if devices[0].ID != onlineID.String() || devices[0].Status != StatusOnline || devices[0].PM25 != 8.5 {
		t.Fatalf("unexpected online device: %#v", devices[0])
	}
	if devices[1].ID != offlineID.String() || devices[1].Status != StatusOffline || !devices[1].IsOutdoor {
		t.Fatalf("unexpected offline device: %#v", devices[1])
	}
}

func TestGetPublicMapDevicesReturnsEmptyArray(t *testing.T) {
	devices, err := NewService(mapRepositoryStub{}).GetPublicMapDevices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if devices == nil || len(devices) != 0 {
		t.Fatalf("got %#v, want non-nil empty slice", devices)
	}
}

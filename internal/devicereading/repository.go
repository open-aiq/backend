package devicereading

import (
	"context"

	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/ent"
)

// Repository defines the data access interface for device readings.
type Repository interface {
	Create(ctx context.Context, deviceID uuid.UUID, req *UploadRequest) (*ent.DeviceReading, error)
}

// entRepository is an Ent-backed implementation of Repository.
type entRepository struct {
	client *ent.Client
}

// NewEntRepository creates a device reading repository backed by the Ent client.
func NewEntRepository(client *ent.Client) Repository {
	return &entRepository{client: client}
}

func (r *entRepository) Create(ctx context.Context, deviceID uuid.UUID, req *UploadRequest) (*ent.DeviceReading, error) {
	create := r.client.DeviceReading.
		Create().
		SetDeviceID(deviceID).
		SetPm10(*req.PMSData.PM10).
		SetPm25(*req.PMSData.PM25).
		SetPm100(*req.PMSData.PM100).
		SetPmsProvider(req.PMSData.Provider).
		SetAqi(*req.AQI).
		SetTemperature(*req.TemperatureData.Temperature).
		SetHumidity(*req.TemperatureData.Humidity).
		SetHeatIndex(*req.TemperatureData.HeatIndex).
		SetTemperatureProvider(req.TemperatureData.Provider)

	if req.Location != nil {
		create.
			SetLat(*req.Location.Lat).
			SetLon(*req.Location.Lon).
			SetLocationProvider(req.Location.Provider)
	}

	return create.Save(ctx)
}

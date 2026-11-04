package devicelog

import (
	"context"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/ent"
	entdevice "go-aiq-backend/internal/platform/ent/device"
	entlog "go-aiq-backend/internal/platform/ent/devicelog"
)

// NewLog is a validated entry ready to store.
type NewLog struct {
	Level     entlog.Level
	Tag       string
	Message   string
	UptimeMs  int64
	BootCount int
	DeviceTs  *time.Time
}

// Repository defines data access for device logs.
type Repository interface {
	CreateBatch(ctx context.Context, deviceID uuid.UUID, logs []NewLog) error
	// DeviceOwned reports whether ownerID owns the device.
	DeviceOwned(ctx context.Context, ownerID string, deviceID uuid.UUID) (bool, error)
	// List returns up to limit logs at or above the given levels, newest first, with id < before (0 = no cursor).
	List(ctx context.Context, deviceID uuid.UUID, levels []entlog.Level, before int64, limit int) ([]*ent.DeviceLog, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error)
}

type entRepository struct {
	client *ent.Client
}

// NewEntRepository creates a device log repository backed by the Ent client.
func NewEntRepository(client *ent.Client) Repository {
	return &entRepository{client: client}
}

func (r *entRepository) CreateBatch(ctx context.Context, deviceID uuid.UUID, logs []NewLog) error {
	builders := make([]*ent.DeviceLogCreate, 0, len(logs))
	for _, log := range logs {
		builders = append(builders, r.client.DeviceLog.Create().
			SetDeviceID(deviceID).
			SetLevel(log.Level).
			SetTag(log.Tag).
			SetMessage(log.Message).
			SetUptimeMs(log.UptimeMs).
			SetBootCount(log.BootCount).
			SetNillableDeviceTs(log.DeviceTs))
	}
	return r.client.DeviceLog.CreateBulk(builders...).Exec(ctx)
}

func (r *entRepository) DeviceOwned(ctx context.Context, ownerID string, deviceID uuid.UUID) (bool, error) {
	return r.client.Device.Query().Where(entdevice.ID(deviceID), entdevice.OwnerID(ownerID)).Exist(ctx)
}

func (r *entRepository) List(ctx context.Context, deviceID uuid.UUID, levels []entlog.Level, before int64, limit int) ([]*ent.DeviceLog, error) {
	query := r.client.DeviceLog.Query().Where(entlog.DeviceID(deviceID), entlog.LevelIn(levels...))
	if before > 0 {
		query = query.Where(entlog.IDLT(before))
	}
	return query.Order(entlog.ByID(entsql.OrderDesc())).Limit(limit).All(ctx)
}

func (r *entRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	return r.client.DeviceLog.Delete().Where(entlog.CreatedAtLT(cutoff)).Exec(ctx)
}

package firmware

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/ent"
	entdevice "go-aiq-backend/internal/platform/ent/device"
	"go-aiq-backend/internal/platform/ent/firmwarerelease"
	"go-aiq-backend/internal/platform/ent/otajob"
)

// JobUpdate holds the job fields to change; nil fields are left unchanged.
type JobUpdate struct {
	Status     *otajob.Status
	Progress   *int
	Error      *string
	ClearError bool
	ReleaseID  *uuid.UUID
	ApprovedBy *string
}

// Repository defines data access for firmware releases and OTA jobs.
type Repository interface {
	// CreateRelease stores a release; a duplicate version is an ent constraint error.
	CreateRelease(ctx context.Context, req PublishReleaseRequest) (*ent.FirmwareRelease, error)
	ListReleases(ctx context.Context) ([]*ent.FirmwareRelease, error)
	GetRelease(ctx context.Context, id uuid.UUID) (*ent.FirmwareRelease, error)
	// GetOwnedDevice returns an ent.NotFoundError unless ownerID owns the device.
	GetOwnedDevice(ctx context.Context, ownerID string, id uuid.UUID) (*ent.Device, error)
	GetDevice(ctx context.Context, id uuid.UUID) (*ent.Device, error)
	// LatestJob returns the newest job of a device (release loaded), or nil.
	LatestJob(ctx context.Context, deviceID uuid.UUID) (*ent.OtaJob, error)
	// LatestJobs returns the newest job per device (release loaded).
	LatestJobs(ctx context.Context, deviceIDs []uuid.UUID) (map[uuid.UUID]*ent.OtaJob, error)
	CreateJob(ctx context.Context, deviceID, releaseID uuid.UUID, approvedBy string) (*ent.OtaJob, error)
	UpdateJob(ctx context.Context, id uuid.UUID, update JobUpdate) (*ent.OtaJob, error)
}

type entRepository struct {
	client *ent.Client
}

// NewEntRepository creates a firmware repository backed by the Ent client.
func NewEntRepository(client *ent.Client) Repository {
	return &entRepository{client: client}
}

func (r *entRepository) CreateRelease(ctx context.Context, req PublishReleaseRequest) (*ent.FirmwareRelease, error) {
	return r.client.FirmwareRelease.Create().
		SetVersion(req.Version).
		SetURL(req.URL).
		SetSha256(req.SHA256).
		SetSize(req.Size).
		SetNotes(req.Notes).
		Save(ctx)
}

func (r *entRepository) ListReleases(ctx context.Context) ([]*ent.FirmwareRelease, error) {
	return r.client.FirmwareRelease.Query().
		Order(firmwarerelease.ByCreatedAt(entsql.OrderDesc())).
		All(ctx)
}

func (r *entRepository) GetRelease(ctx context.Context, id uuid.UUID) (*ent.FirmwareRelease, error) {
	return r.client.FirmwareRelease.Get(ctx, id)
}

func (r *entRepository) GetOwnedDevice(ctx context.Context, ownerID string, id uuid.UUID) (*ent.Device, error) {
	return r.client.Device.Query().Where(entdevice.ID(id), entdevice.OwnerID(ownerID)).Only(ctx)
}

func (r *entRepository) GetDevice(ctx context.Context, id uuid.UUID) (*ent.Device, error) {
	return r.client.Device.Get(ctx, id)
}

func (r *entRepository) LatestJob(ctx context.Context, deviceID uuid.UUID) (*ent.OtaJob, error) {
	job, err := r.client.OtaJob.Query().
		Where(otajob.DeviceID(deviceID)).
		WithRelease().
		Order(otajob.ByCreatedAt(entsql.OrderDesc())).
		First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	return job, err
}

func (r *entRepository) LatestJobs(ctx context.Context, deviceIDs []uuid.UUID) (map[uuid.UUID]*ent.OtaJob, error) {
	jobs, err := r.client.OtaJob.Query().
		Where(otajob.DeviceIDIn(deviceIDs...)).
		WithRelease().
		Order(otajob.ByCreatedAt(entsql.OrderDesc())).
		All(ctx)
	if err != nil {
		return nil, err
	}
	latest := make(map[uuid.UUID]*ent.OtaJob, len(deviceIDs))
	for _, job := range jobs {
		if _, seen := latest[job.DeviceID]; !seen {
			latest[job.DeviceID] = job
		}
	}
	return latest, nil
}

func (r *entRepository) CreateJob(ctx context.Context, deviceID, releaseID uuid.UUID, approvedBy string) (*ent.OtaJob, error) {
	job, err := r.client.OtaJob.Create().
		SetDeviceID(deviceID).
		SetReleaseID(releaseID).
		SetApprovedBy(approvedBy).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.client.OtaJob.Query().Where(otajob.ID(job.ID)).WithRelease().Only(ctx)
}

func (r *entRepository) UpdateJob(ctx context.Context, id uuid.UUID, update JobUpdate) (*ent.OtaJob, error) {
	builder := r.client.OtaJob.UpdateOneID(id)
	if update.Status != nil {
		builder.SetStatus(*update.Status)
	}
	if update.Progress != nil {
		builder.SetProgress(*update.Progress)
	}
	if update.Error != nil {
		builder.SetError(*update.Error)
	} else if update.ClearError {
		builder.ClearError()
	}
	if update.ReleaseID != nil {
		builder.SetReleaseID(*update.ReleaseID)
	}
	if update.ApprovedBy != nil {
		builder.SetApprovedBy(*update.ApprovedBy).SetApprovedAt(timeNow())
	}
	if err := builder.Exec(ctx); err != nil {
		return nil, err
	}
	return r.client.OtaJob.Query().Where(otajob.ID(id)).WithRelease().Only(ctx)
}

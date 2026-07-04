package device

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/ent"
	entdevice "go-aiq-backend/internal/platform/ent/device"
)

// Repository defines the data access interface for devices.
type Repository interface {
	// Create stores a new device. deviceKeyHash is the SHA-256 digest of the
	// secret key, never the raw key.
	Create(ctx context.Context, deviceID, name, deviceKeyHash string, isOutdoor, isPublic bool) (*ent.Device, error)
	List(ctx context.Context) ([]*ent.Device, error)
	// GetByDeviceID returns the device with the given public device_id. It
	// returns an ent.NotFoundError (see ent.IsNotFound) if no such device exists.
	GetByDeviceID(ctx context.Context, deviceID string) (*ent.Device, error)
	// Update applies the non-nil fields of req to the device with the given
	// internal id. It returns an ent.NotFoundError (see ent.IsNotFound) if no
	// such device exists.
	Update(ctx context.Context, id uuid.UUID, req UpdateDeviceRequest) (*ent.Device, error)
	// UpdateKey replaces the stored key digest of the device with the given
	// internal id. It returns an ent.NotFoundError (see ent.IsNotFound) if no
	// such device exists.
	UpdateKey(ctx context.Context, id uuid.UUID, deviceKeyHash string) (*ent.Device, error)
	// Delete removes the device with the given internal id. It returns an
	// ent.NotFoundError (see ent.IsNotFound) if no such device exists.
	Delete(ctx context.Context, id uuid.UUID) error
}

// entRepository is an Ent-backed implementation of Repository.
type entRepository struct {
	client *ent.Client
}

// NewEntRepository creates a device repository backed by the Ent client.
func NewEntRepository(client *ent.Client) Repository {
	return &entRepository{client: client}
}

func (r *entRepository) Create(ctx context.Context, deviceID, name, deviceKeyHash string, isOutdoor, isPublic bool) (*ent.Device, error) {
	return r.client.Device.
		Create().
		SetDeviceID(deviceID).
		SetName(name).
		SetDeviceKey(deviceKeyHash).
		SetIsOutdoor(isOutdoor).
		SetIsPublic(isPublic).
		Save(ctx)
}

func (r *entRepository) List(ctx context.Context) ([]*ent.Device, error) {
	// Newest devices first.
	return r.client.Device.
		Query().
		Order(entdevice.ByCreatedAt(entsql.OrderDesc())).
		All(ctx)
}

func (r *entRepository) GetByDeviceID(ctx context.Context, deviceID string) (*ent.Device, error) {
	return r.client.Device.
		Query().
		Where(entdevice.DeviceID(deviceID)).
		Only(ctx)
}

func (r *entRepository) Update(ctx context.Context, id uuid.UUID, req UpdateDeviceRequest) (*ent.Device, error) {
	return r.client.Device.
		UpdateOneID(id).
		SetNillableName(req.Name).
		SetNillableIsOutdoor(req.IsOutdoor).
		SetNillableIsPublic(req.IsPublic).
		Save(ctx)
}

func (r *entRepository) UpdateKey(ctx context.Context, id uuid.UUID, deviceKeyHash string) (*ent.Device, error) {
	return r.client.Device.
		UpdateOneID(id).
		SetDeviceKey(deviceKeyHash).
		Save(ctx)
}

func (r *entRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Device.DeleteOneID(id).Exec(ctx)
}

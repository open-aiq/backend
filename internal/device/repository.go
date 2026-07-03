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
	Create(ctx context.Context, deviceID, name, deviceKey string) (*ent.Device, error)
	List(ctx context.Context) ([]*ent.Device, error)
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

func (r *entRepository) Create(ctx context.Context, deviceID, name, deviceKey string) (*ent.Device, error) {
	return r.client.Device.
		Create().
		SetDeviceID(deviceID).
		SetName(name).
		SetDeviceKey(deviceKey).
		Save(ctx)
}

func (r *entRepository) List(ctx context.Context) ([]*ent.Device, error) {
	// Newest devices first.
	return r.client.Device.
		Query().
		Order(entdevice.ByCreatedAt(entsql.OrderDesc())).
		All(ctx)
}

func (r *entRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Device.DeleteOneID(id).Exec(ctx)
}

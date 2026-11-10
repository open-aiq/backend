package account

import (
	"context"

	"go-aiq-backend/internal/platform/ent"
	entdevice "go-aiq-backend/internal/platform/ent/device"
)

// Repository defines the data access interface for an account's data.
type Repository interface {
	// DeleteOwnedDevices removes every device the user owns and returns how
	// many it removed. Readings, logs and OTA jobs go with them through the
	// database's ON DELETE CASCADE.
	DeleteOwnedDevices(ctx context.Context, ownerID string) (int, error)
}

// entRepository is an Ent-backed implementation of Repository.
type entRepository struct {
	client *ent.Client
}

// NewEntRepository creates an account repository backed by the Ent client.
func NewEntRepository(client *ent.Client) Repository {
	return &entRepository{client: client}
}

func (r *entRepository) DeleteOwnedDevices(ctx context.Context, ownerID string) (int, error) {
	return r.client.Device.Delete().Where(entdevice.OwnerID(ownerID)).Exec(ctx)
}

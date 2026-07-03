package device

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/ent"
)

// ErrDeviceNotFound is returned when an operation targets a device that does not exist.
var ErrDeviceNotFound = errors.New("device not found")

const (
	// deviceIDPrefix namespaces the public device identifier.
	deviceIDPrefix = "dev_"
	// deviceKeyPrefix marks the value as a secret key (secret-scanner friendly).
	deviceKeyPrefix = "sk_"
	// deviceKeyBytes is the amount of entropy in a device key.
	deviceKeyBytes = 32
)

// Service handles device business logic.
type Service struct {
	repo Repository
}

// NewService creates a new device service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create registers a new device, generating its public device_id and secret
// device_key. The returned CreatedDevice contains the key, which is the only
// time it is ever exposed.
func (s *Service) Create(ctx context.Context, name string) (*CreatedDevice, error) {
	deviceID := deviceIDPrefix + strings.ReplaceAll(uuid.New().String(), "-", "")

	deviceKey, err := generateDeviceKey()
	if err != nil {
		return nil, fmt.Errorf("generate device key: %w", err)
	}

	created, err := s.repo.Create(ctx, deviceID, name, deviceKey)
	if err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}

	return &CreatedDevice{
		ID:        created.ID,
		DeviceID:  created.DeviceID,
		Name:      created.Name,
		DeviceKey: deviceKey,
		CreatedAt: created.CreatedAt,
		UpdatedAt: created.UpdatedAt,
	}, nil
}

// Delete removes a device by its internal id. It returns ErrDeviceNotFound if no
// device with that id exists.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if ent.IsNotFound(err) {
			return ErrDeviceNotFound
		}
		return fmt.Errorf("delete device: %w", err)
	}
	return nil
}

// List returns all registered devices without their secret keys.
func (s *Service) List(ctx context.Context) ([]Device, error) {
	entDevices, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}

	devices := make([]Device, 0, len(entDevices))
	for _, d := range entDevices {
		devices = append(devices, toDevice(d))
	}
	return devices, nil
}

// toDevice maps an Ent entity to the public device view (no secret key).
func toDevice(d *ent.Device) Device {
	return Device{
		ID:        d.ID,
		DeviceID:  d.DeviceID,
		Name:      d.Name,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// generateDeviceKey returns a cryptographically random, prefixed secret key.
func generateDeviceKey() (string, error) {
	buf := make([]byte, deviceKeyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return deviceKeyPrefix + hex.EncodeToString(buf), nil
}

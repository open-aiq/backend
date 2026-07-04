package device

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/ent"
)

// ErrDeviceNotFound is returned when an operation targets a device that does not exist.
var ErrDeviceNotFound = errors.New("device not found")

// ErrInvalidCredentials is returned when a device_id/device_key pair does not
// authenticate. It deliberately covers both "unknown device" and "wrong key" so
// device ids cannot be probed.
var ErrInvalidCredentials = errors.New("invalid device credentials")

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
func (s *Service) Create(ctx context.Context, req CreateDeviceRequest) (*CreatedDevice, error) {
	deviceID := deviceIDPrefix + strings.ReplaceAll(uuid.New().String(), "-", "")

	deviceKey, err := generateDeviceKey()
	if err != nil {
		return nil, fmt.Errorf("generate device key: %w", err)
	}

	created, err := s.repo.Create(ctx, deviceID, req.Name, hashDeviceKey(deviceKey), req.IsOutdoor, req.IsPublic)
	if err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}

	return &CreatedDevice{
		ID:        created.ID,
		DeviceID:  created.DeviceID,
		Name:      created.Name,
		IsOutdoor: created.IsOutdoor,
		IsPublic:  created.IsPublic,
		DeviceKey: deviceKey,
		CreatedAt: created.CreatedAt,
		UpdatedAt: created.UpdatedAt,
	}, nil
}

// ErrNoUpdateFields is returned when an update request contains no fields.
var ErrNoUpdateFields = errors.New("no fields to update")

// Update applies a partial update (name, is_outdoor, is_public) to a device
// and returns its updated public view. It returns ErrDeviceNotFound if no
// device with that id exists, and ErrNoUpdateFields if the request is empty.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateDeviceRequest) (*Device, error) {
	if req.Name == nil && req.IsOutdoor == nil && req.IsPublic == nil {
		return nil, ErrNoUpdateFields
	}

	updated, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("update device: %w", err)
	}

	d := toDevice(updated)
	return &d, nil
}

// RotateKey generates a new secret key for the device with the given internal
// id, invalidating the old key immediately. As at creation, the returned
// CreatedDevice is the only time the new key is ever exposed. It returns
// ErrDeviceNotFound if no device with that id exists.
func (s *Service) RotateKey(ctx context.Context, id uuid.UUID) (*CreatedDevice, error) {
	deviceKey, err := generateDeviceKey()
	if err != nil {
		return nil, fmt.Errorf("generate device key: %w", err)
	}

	updated, err := s.repo.UpdateKey(ctx, id, hashDeviceKey(deviceKey))
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("rotate device key: %w", err)
	}

	return &CreatedDevice{
		ID:        updated.ID,
		DeviceID:  updated.DeviceID,
		Name:      updated.Name,
		IsOutdoor: updated.IsOutdoor,
		IsPublic:  updated.IsPublic,
		DeviceKey: deviceKey,
		CreatedAt: updated.CreatedAt,
		UpdatedAt: updated.UpdatedAt,
	}, nil
}

// Authenticate verifies a device_id/device_key pair and returns the device's
// internal id on success, or ErrInvalidCredentials otherwise.
func (s *Service) Authenticate(ctx context.Context, deviceID, deviceKey string) (uuid.UUID, error) {
	d, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.Nil, ErrInvalidCredentials
		}
		return uuid.Nil, fmt.Errorf("get device: %w", err)
	}

	if subtle.ConstantTimeCompare([]byte(d.DeviceKey), []byte(hashDeviceKey(deviceKey))) != 1 {
		return uuid.Nil, ErrInvalidCredentials
	}
	return d.ID, nil
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
		IsOutdoor: d.IsOutdoor,
		IsPublic:  d.IsPublic,
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

// hashDeviceKey returns the hex SHA-256 digest of a device key; only digests
// are stored. Keys carry 32 bytes of random entropy, so a fast hash (rather
// than bcrypt/argon2) is sufficient and keeps ingest auth cheap.
func hashDeviceKey(deviceKey string) string {
	sum := sha256.Sum256([]byte(deviceKey))
	return hex.EncodeToString(sum[:])
}

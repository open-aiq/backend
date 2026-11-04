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
	"go-aiq-backend/internal/platform/middleware"
)

// ErrDeviceNotFound is returned when an operation targets a device that does not exist.
var ErrDeviceNotFound = errors.New("device not found")

// ErrInvalidCredentials is returned when a device_id/device_key pair does not
// authenticate. It deliberately covers both "unknown device" and "wrong key" so
// device ids cannot be probed. It is the middleware's sentinel so DeviceAuth
// can recognize it without importing this package.
var ErrInvalidCredentials = middleware.ErrInvalidDeviceCredentials

const (
	// deviceIDPrefix namespaces the public device identifier.
	deviceIDPrefix = "dev_"
	// deviceKeyPrefix marks the value as a secret key (secret-scanner friendly).
	deviceKeyPrefix = "sk_"
	// deviceKeyBytes is the amount of entropy in a device key.
	deviceKeyBytes = 32
)

// OtaSummarizer reports the firmware-update state of devices for listings.
// Implemented by firmware.Service; optional.
type OtaSummarizer interface {
	SummarizeOta(ctx context.Context, devices []*ent.Device) (map[uuid.UUID]OtaSummary, error)
}

// Service handles device business logic.
type Service struct {
	repo Repository
	ota  OtaSummarizer
}

// NewService creates a new device service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetOtaSummarizer enables the per-device "ota" summary in listings.
func (s *Service) SetOtaSummarizer(ota OtaSummarizer) {
	s.ota = ota
}

// Create registers a new device, generating its public device_id and secret
// device_key. The returned CreatedDevice contains the key, which is the only
// time it is ever exposed.
func (s *Service) Create(ctx context.Context, ownerID string, req CreateDeviceRequest) (*CreatedDevice, error) {
	if req.IsLocationPublic && !req.IsPublic {
		return nil, ErrLocationRequiresPublic
	}
	deviceID := deviceIDPrefix + strings.ReplaceAll(uuid.New().String(), "-", "")

	deviceKey, err := generateDeviceKey()
	if err != nil {
		return nil, fmt.Errorf("generate device key: %w", err)
	}

	created, err := s.repo.Create(ctx, ownerID, deviceID, req.Name, hashDeviceKey(deviceKey), req.IsOutdoor, req.IsPublic, req.IsLocationPublic)
	if err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}

	return &CreatedDevice{
		ID:               created.ID,
		DeviceID:         created.DeviceID,
		Name:             created.Name,
		IsOutdoor:        created.IsOutdoor,
		IsPublic:         created.IsPublic,
		IsLocationPublic: created.IsLocationPublic,
		DeviceKey:        deviceKey,
		CreatedAt:        created.CreatedAt,
		UpdatedAt:        created.UpdatedAt,
	}, nil
}

// ErrNoUpdateFields is returned when an update request contains no fields.
var ErrNoUpdateFields = errors.New("no fields to update")
var ErrLocationRequiresPublic = errors.New("location sharing requires a public device")

// Update applies a partial update to a device, enforcing explicit location
// consent and revoking it whenever the device becomes private.
// and returns its updated public view. It returns ErrDeviceNotFound if no
// device with that id exists, and ErrNoUpdateFields if the request is empty.
func (s *Service) Update(ctx context.Context, ownerID string, id uuid.UUID, req UpdateDeviceRequest) (*Device, error) {
	if req.Name == nil && req.IsOutdoor == nil && req.IsPublic == nil && req.IsLocationPublic == nil {
		return nil, ErrNoUpdateFields
	}
	if req.IsPublic != nil && !*req.IsPublic {
		locationPrivate := false
		req.IsLocationPublic = &locationPrivate
	}
	if req.IsLocationPublic != nil && *req.IsLocationPublic {
		isPublic := req.IsPublic != nil && *req.IsPublic
		if req.IsPublic == nil {
			existing, err := s.repo.Get(ctx, ownerID, id)
			if err != nil {
				if ent.IsNotFound(err) {
					return nil, ErrDeviceNotFound
				}
				return nil, fmt.Errorf("get device: %w", err)
			}
			isPublic = existing.IsPublic
		}
		if !isPublic {
			return nil, ErrLocationRequiresPublic
		}
	}

	updated, err := s.repo.Update(ctx, ownerID, id, req)
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
func (s *Service) RotateKey(ctx context.Context, ownerID string, id uuid.UUID) (*CreatedDevice, error) {
	deviceKey, err := generateDeviceKey()
	if err != nil {
		return nil, fmt.Errorf("generate device key: %w", err)
	}

	updated, err := s.repo.UpdateKey(ctx, ownerID, id, hashDeviceKey(deviceKey))
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("rotate device key: %w", err)
	}

	return &CreatedDevice{
		ID:               updated.ID,
		DeviceID:         updated.DeviceID,
		Name:             updated.Name,
		IsOutdoor:        updated.IsOutdoor,
		IsPublic:         updated.IsPublic,
		IsLocationPublic: updated.IsLocationPublic,
		DeviceKey:        deviceKey,
		CreatedAt:        updated.CreatedAt,
		UpdatedAt:        updated.UpdatedAt,
	}, nil
}

// Authenticate verifies a device_id/device_key pair and returns the device's
// internal id on success, or ErrInvalidCredentials otherwise.
func (s *Service) Authenticate(ctx context.Context, deviceID, deviceKey string) (uuid.UUID, error) {
	d, err := s.authenticate(ctx, deviceID, deviceKey)
	if err != nil {
		return uuid.Nil, err
	}
	return d.ID, nil
}

// AuthenticateDevice authenticates a device (see Authenticate) and records the
// firmware version it reported when that changed. An empty version is ignored.
// Implements middleware.DeviceAuthenticator.
func (s *Service) AuthenticateDevice(ctx context.Context, deviceID, deviceKey, firmwareVersion string) (uuid.UUID, error) {
	d, err := s.authenticate(ctx, deviceID, deviceKey)
	if err != nil {
		return uuid.Nil, err
	}
	if firmwareVersion != "" && (d.FirmwareVersion == nil || *d.FirmwareVersion != firmwareVersion) {
		if err := s.repo.SetFirmwareVersion(ctx, d.ID, firmwareVersion); err != nil {
			return uuid.Nil, fmt.Errorf("record firmware version: %w", err)
		}
	}
	return d.ID, nil
}

func (s *Service) authenticate(ctx context.Context, deviceID, deviceKey string) (*ent.Device, error) {
	d, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get device: %w", err)
	}

	if subtle.ConstantTimeCompare([]byte(d.DeviceKey), []byte(hashDeviceKey(deviceKey))) != 1 {
		return nil, ErrInvalidCredentials
	}
	return d, nil
}

// Delete removes a device by its internal id. It returns ErrDeviceNotFound if no
// device with that id exists.
func (s *Service) Delete(ctx context.Context, ownerID string, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, ownerID, id); err != nil {
		if ent.IsNotFound(err) {
			return ErrDeviceNotFound
		}
		return fmt.Errorf("delete device: %w", err)
	}
	return nil
}

// List returns all registered devices without their secret keys.
func (s *Service) List(ctx context.Context, ownerID string) ([]Device, error) {
	entDevices, err := s.repo.List(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}

	var summaries map[uuid.UUID]OtaSummary
	if s.ota != nil && len(entDevices) > 0 {
		summaries, err = s.ota.SummarizeOta(ctx, entDevices)
		if err != nil {
			return nil, fmt.Errorf("summarize ota: %w", err)
		}
	}

	devices := make([]Device, 0, len(entDevices))
	for _, d := range entDevices {
		device := toDevice(d)
		if summary, ok := summaries[d.ID]; ok {
			device.Ota = &summary
		}
		devices = append(devices, device)
	}
	return devices, nil
}

func (s *Service) ListPublic(ctx context.Context) ([]PublicDevice, error) {
	rows, err := s.repo.ListPublic(ctx)
	if err != nil {
		return nil, fmt.Errorf("list public devices: %w", err)
	}
	out := make([]PublicDevice, 0, len(rows))
	for _, d := range rows {
		out = append(out, toPublicDevice(d))
	}
	return out, nil
}

func (s *Service) GetPublic(ctx context.Context, id uuid.UUID) (*PublicDevice, error) {
	d, err := s.repo.GetPublic(ctx, id)
	if ent.IsNotFound(err) {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, err
	}
	p := toPublicDevice(d)
	return &p, nil
}

func toPublicDevice(d *ent.Device) PublicDevice {
	return PublicDevice{ID: d.ID, Name: d.Name, IsOutdoor: d.IsOutdoor, IsPublic: d.IsPublic, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

// toDevice maps an Ent entity to the public device view (no secret key).
func toDevice(d *ent.Device) Device {
	return Device{
		ID:               d.ID,
		DeviceID:         d.DeviceID,
		Name:             d.Name,
		IsOutdoor:        d.IsOutdoor,
		IsPublic:         d.IsPublic,
		IsLocationPublic: d.IsLocationPublic,
		FirmwareVersion:  d.FirmwareVersion,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
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

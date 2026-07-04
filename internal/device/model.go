package device

import (
	"time"

	"github.com/google/uuid"
)

// CreateDeviceRequest is the payload for registering a new device. The
// booleans default to false when omitted.
type CreateDeviceRequest struct {
	Name      string `json:"name" binding:"required" example:"Living Room Sensor"`
	IsOutdoor bool   `json:"is_outdoor" example:"false"`
	IsPublic  bool   `json:"is_public" example:"false"`
}

// UpdateDeviceRequest is the payload for partially updating a device. Only the
// provided fields are changed.
type UpdateDeviceRequest struct {
	Name      *string `json:"name,omitempty" binding:"omitempty,min=1" example:"Balcony Sensor"`
	IsOutdoor *bool   `json:"is_outdoor,omitempty"`
	IsPublic  *bool   `json:"is_public,omitempty"`
}

// Device is the public view of a registered device (no secret key). "id" is the
// resource identifier used by the management API (list, delete); "device_id" and
// the secret key are the credentials the physical device uses to send data.
type Device struct {
	ID        uuid.UUID `json:"id"`
	DeviceID  string    `json:"device_id" example:"dev_9f1c8a3b4d2e4f0a5b6c7d8e9f001122"`
	Name      string    `json:"name" example:"Living Room Sensor"`
	IsOutdoor bool      `json:"is_outdoor" example:"false"`
	IsPublic  bool      `json:"is_public" example:"false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreatedDevice is returned only when a device is created or its key is
// rotated. It includes the secret device_key, which is shown exactly once and
// never listed again.
type CreatedDevice struct {
	ID        uuid.UUID `json:"id"`
	DeviceID  string    `json:"device_id" example:"dev_9f1c8a3b4d2e4f0a5b6c7d8e9f001122"`
	Name      string    `json:"name" example:"Living Room Sensor"`
	IsOutdoor bool      `json:"is_outdoor" example:"false"`
	IsPublic  bool      `json:"is_public" example:"false"`
	DeviceKey string    `json:"device_key" example:"sk_1a2b3c4d5e6f...store this now, it won't be shown again"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Response types for swagger documentation.

type CreateDeviceResponse struct {
	Data CreatedDevice `json:"data"`
}

type ListDevicesResponse struct {
	Data []Device `json:"data"`
}

type RotateKeyResponse struct {
	Data CreatedDevice `json:"data"`
}

type UpdateDeviceResponse struct {
	Data Device `json:"data"`
}

type ErrorResponse struct {
	Error   string `json:"error" example:"Invalid request body"`
	Details string `json:"details,omitempty" example:"name is required"`
}

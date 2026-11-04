package devicelog

import "time"

// MaxBatch is the most entries a device may upload per request.
const MaxBatch = 50

// Entry is one log record uploaded by a device.
type Entry struct {
	Level     string `json:"level" validate:"required,oneof=debug info warn error" example:"warn"`
	Tag       string `json:"tag" mod:"trim" validate:"max=32" example:"wifi"`
	Message   string `json:"message" mod:"trim" validate:"required,max=256" example:"reconnect failed: timeout"`
	UptimeMs  *int64 `json:"uptime_ms" validate:"required,gte=0" example:"183000"`
	BootCount *int   `json:"boot_count" validate:"required,gte=0" example:"12"`
	// Ts is the device's Unix time in seconds, omitted until its clock is synced.
	Ts *int64 `json:"ts" validate:"omitempty,gt=0" example:"1793700000"`
}

// UploadRequest is the payload of POST /device/logs.
type UploadRequest struct {
	Entries []Entry `json:"entries" validate:"max=50,dive"`
	// Dropped counts entries the device discarded because its buffer was full.
	Dropped int `json:"dropped" validate:"gte=0" example:"0"`
}

// UploadResult acknowledges an upload.
type UploadResult struct {
	Accepted int `json:"accepted" example:"12"`
}

// Log is one stored entry as returned to the owner.
type Log struct {
	ID        int64      `json:"id"`
	Level     string     `json:"level" example:"warn"`
	Tag       string     `json:"tag" example:"wifi"`
	Message   string     `json:"message"`
	UptimeMs  int64      `json:"uptime_ms"`
	BootCount int        `json:"boot_count"`
	DeviceTs  *time.Time `json:"device_ts"`
	CreatedAt time.Time  `json:"created_at"`
}

// ListQuery filters a device's logs. Level is the minimum level returned.
type ListQuery struct {
	Level  string `form:"level" validate:"omitempty,oneof=debug info warn error"`
	Limit  int    `form:"limit" validate:"omitempty,gte=1,lte=200"`
	Before int64  `form:"before" validate:"omitempty,gt=0"`
}

// Response types for swagger documentation.

type UploadResponse struct {
	Data UploadResult `json:"data"`
}

type ListResponse struct {
	Data []Log `json:"data"`
	// NextBefore is the cursor for the next (older) page, or null at the end.
	NextBefore *int64 `json:"next_before"`
}

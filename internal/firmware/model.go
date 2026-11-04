package firmware

import (
	"time"

	"github.com/google/uuid"
)

// MaxImageSize is the size of one OTA app slot in the device partition table
// (device/partitions.csv: 0x1F0000). Larger images could never be installed.
const MaxImageSize int64 = 0x1F0000

// PublishReleaseRequest registers a firmware image built and hosted by CI.
type PublishReleaseRequest struct {
	Version string `json:"version" mod:"trim" validate:"required,max=32" example:"0.2.0"`
	URL     string `json:"url" mod:"trim" validate:"required,url,startswith=https://,max=2048" example:"https://github.com/open-aiq/aqs-device/releases/download/v0.2.0/device_rust.bin"`
	SHA256  string `json:"sha256" mod:"trim,lcase" validate:"required,len=64,hexadecimal" example:"53ff9481b8ea2e8ec2c9533432c1e66df99c68a0a9e3079dc455547b13d39346"`
	Size    int64  `json:"size" validate:"required,gt=0" example:"1600800"`
	Notes   string `json:"notes" mod:"trim" validate:"max=4000" example:"Adds secure BLE pairing."`
}

// Release is a published firmware image.
type Release struct {
	ID        uuid.UUID `json:"id"`
	Version   string    `json:"version" example:"0.2.0"`
	URL       string    `json:"url"`
	SHA256    string    `json:"sha256"`
	Size      int64     `json:"size" example:"1600800"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// Job is an owner-approved update of one device to one release.
type Job struct {
	ID         uuid.UUID `json:"id"`
	ReleaseID  uuid.UUID `json:"release_id"`
	Version    string    `json:"version" example:"0.2.0"`
	Status     string    `json:"status" example:"downloading" enums:"approved,downloading,installing,succeeded,failed,cancelled"`
	Progress   int       `json:"progress" example:"40"`
	Error      *string   `json:"error"`
	ApprovedAt time.Time `json:"approved_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// DeviceOta is the update state of one device for its owner.
type DeviceOta struct {
	// CurrentVersion is the firmware the device last reported; null if it never did
	// (firmware that predates OTA support).
	CurrentVersion *string `json:"current_version" example:"0.1.0"`
	// Available is the newest release newer than CurrentVersion, if any.
	Available *Release `json:"available"`
	// Job is the most recent update job of the device, if any.
	Job *Job `json:"job"`
}

// ApproveRequest approves updating a device to a release.
type ApproveRequest struct {
	ReleaseID string `json:"release_id" validate:"required,uuid4"`
}

// DeviceJob is what a device receives from GET /device/ota.
type DeviceJob struct {
	JobID   uuid.UUID `json:"job_id"`
	Version string    `json:"version" example:"0.2.0"`
	URL     string    `json:"url"`
	SHA256  string    `json:"sha256"`
	Size    int64     `json:"size"`
	Status  string    `json:"status" example:"approved" enums:"approved,downloading,installing"`
}

// StatusReportRequest is a device's progress report for a job.
type StatusReportRequest struct {
	JobID    string  `json:"job_id" validate:"required,uuid4"`
	Status   string  `json:"status" validate:"required,oneof=downloading installing succeeded failed" example:"downloading"`
	Progress *int    `json:"progress" validate:"omitempty,gte=0,lte=100" example:"40"`
	Error    *string `json:"error" mod:"trim" validate:"omitempty,max=500"`
	// Version is the firmware version currently running on the device.
	Version string `json:"version" mod:"trim" validate:"required,max=32" example:"0.1.0"`
}

// Response types for swagger documentation.

type ReleaseResponse struct {
	Data Release `json:"data"`
}

type DeviceOtaResponse struct {
	Data DeviceOta `json:"data"`
}

type JobResponse struct {
	Data Job `json:"data"`
}

// DeviceJobResponse carries the pending job, or null when there is none.
type DeviceJobResponse struct {
	Data *DeviceJob `json:"data"`
}

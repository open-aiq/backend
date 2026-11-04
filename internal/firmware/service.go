package firmware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/mod/semver"

	"go-aiq-backend/internal/device"
	"go-aiq-backend/internal/platform/ent"
	"go-aiq-backend/internal/platform/ent/otajob"
)

var (
	ErrDeviceNotFound    = errors.New("device not found")
	ErrReleaseNotFound   = errors.New("release not found")
	ErrJobNotFound       = errors.New("ota job not found")
	ErrInvalidVersion    = errors.New("version is not a semantic version")
	ErrImageTooLarge     = errors.New("image does not fit an OTA slot")
	ErrReleaseExists     = errors.New("a release with this version already exists")
	ErrFirmwareUnknown   = errors.New("device has not reported a firmware version")
	ErrNotNewer          = errors.New("release is not newer than the device firmware")
	ErrUpdateInProgress  = errors.New("an update is already downloading or installing")
	ErrNothingToCancel   = errors.New("no approved update to cancel")
	ErrInvalidTransition = errors.New("invalid job status transition")
)

// timeNow is replaced in tests.
var timeNow = time.Now

// Summary statuses shown as the device badge.
const (
	SummaryUpToDate        = "up_to_date"
	SummaryUpdateAvailable = "update_available"
	SummaryFailed          = "failed"
)

// Service handles firmware releases and OTA job orchestration.
type Service struct {
	repo Repository
}

// NewService creates a new firmware service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Publish registers a new firmware release.
func (s *Service) Publish(ctx context.Context, req PublishReleaseRequest) (*Release, error) {
	req.Version = strings.TrimPrefix(req.Version, "v")
	if !semver.IsValid("v" + req.Version) {
		return nil, ErrInvalidVersion
	}
	if req.Size > MaxImageSize {
		return nil, ErrImageTooLarge
	}
	created, err := s.repo.CreateRelease(ctx, req)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, ErrReleaseExists
		}
		return nil, fmt.Errorf("create release: %w", err)
	}
	release := toRelease(created)
	return &release, nil
}

// DeviceOta returns the update state of a device owned by ownerID.
func (s *Service) DeviceOta(ctx context.Context, ownerID string, deviceID uuid.UUID) (*DeviceOta, error) {
	d, err := s.ownedDevice(ctx, ownerID, deviceID)
	if err != nil {
		return nil, err
	}
	latest, err := s.latestRelease(ctx)
	if err != nil {
		return nil, err
	}
	job, err := s.repo.LatestJob(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("latest job: %w", err)
	}
	out := &DeviceOta{CurrentVersion: d.FirmwareVersion}
	if latest != nil && d.FirmwareVersion != nil && newer(latest.Version, *d.FirmwareVersion) {
		release := toRelease(latest)
		out.Available = &release
	}
	if job != nil {
		j := toJob(job)
		out.Job = &j
	}
	return out, nil
}

// Approve lets the device install releaseID on its next check-in.
func (s *Service) Approve(ctx context.Context, ownerID string, deviceID uuid.UUID, releaseID uuid.UUID) (*Job, error) {
	d, err := s.ownedDevice(ctx, ownerID, deviceID)
	if err != nil {
		return nil, err
	}
	release, err := s.repo.GetRelease(ctx, releaseID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrReleaseNotFound
		}
		return nil, fmt.Errorf("get release: %w", err)
	}
	if d.FirmwareVersion == nil {
		return nil, ErrFirmwareUnknown
	}
	if !newer(release.Version, *d.FirmwareVersion) {
		return nil, ErrNotNewer
	}
	current, err := s.repo.LatestJob(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("latest job: %w", err)
	}
	if current != nil {
		switch current.Status {
		case otajob.StatusDownloading, otajob.StatusInstalling:
			return nil, ErrUpdateInProgress
		case otajob.StatusApproved:
			status := otajob.StatusApproved
			updated, err := s.repo.UpdateJob(ctx, current.ID, JobUpdate{Status: &status, ReleaseID: &release.ID, ApprovedBy: &ownerID, ClearError: true})
			if err != nil {
				return nil, fmt.Errorf("update job: %w", err)
			}
			job := toJob(updated)
			return &job, nil
		}
	}
	created, err := s.repo.CreateJob(ctx, d.ID, release.ID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	job := toJob(created)
	return &job, nil
}

// Cancel withdraws an approval the device has not started acting on.
func (s *Service) Cancel(ctx context.Context, ownerID string, deviceID uuid.UUID) (*Job, error) {
	d, err := s.ownedDevice(ctx, ownerID, deviceID)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.LatestJob(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("latest job: %w", err)
	}
	if current == nil || current.Status != otajob.StatusApproved {
		return nil, ErrNothingToCancel
	}
	status := otajob.StatusCancelled
	updated, err := s.repo.UpdateJob(ctx, current.ID, JobUpdate{Status: &status})
	if err != nil {
		return nil, fmt.Errorf("update job: %w", err)
	}
	job := toJob(updated)
	return &job, nil
}

// DeviceCheck returns the job a device should act on, or nil. Installing jobs
// are returned too so a rebooted device can report success or rollback.
func (s *Service) DeviceCheck(ctx context.Context, deviceID uuid.UUID) (*DeviceJob, error) {
	job, err := s.repo.LatestJob(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("latest job: %w", err)
	}
	if job == nil || !active(job.Status) || job.Edges.Release == nil {
		return nil, nil
	}
	release := job.Edges.Release
	return &DeviceJob{
		JobID: job.ID, Version: release.Version, URL: release.URL,
		SHA256: release.Sha256, Size: release.Size, Status: string(job.Status),
	}, nil
}

// allowedTransitions lists the statuses a device may report from each status.
var allowedTransitions = map[otajob.Status][]otajob.Status{
	otajob.StatusApproved:    {otajob.StatusDownloading, otajob.StatusSucceeded, otajob.StatusFailed},
	otajob.StatusDownloading: {otajob.StatusDownloading, otajob.StatusInstalling, otajob.StatusSucceeded, otajob.StatusFailed},
	otajob.StatusInstalling:  {otajob.StatusInstalling, otajob.StatusSucceeded, otajob.StatusFailed},
}

// DeviceReport applies a device's progress report to one of its jobs.
func (s *Service) DeviceReport(ctx context.Context, deviceID uuid.UUID, req StatusReportRequest) (*Job, error) {
	jobID, err := uuid.Parse(req.JobID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	job, err := s.repo.LatestJob(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("latest job: %w", err)
	}
	// Only the device's newest job can progress; older ones are history.
	if job == nil || job.ID != jobID {
		return nil, ErrJobNotFound
	}
	next := otajob.Status(req.Status)
	if !transitionAllowed(job.Status, next) {
		return nil, ErrInvalidTransition
	}
	update := JobUpdate{Status: &next, Progress: req.Progress, Error: req.Error}
	if next == otajob.StatusSucceeded {
		reported := strings.TrimPrefix(req.Version, "v")
		if job.Edges.Release == nil || reported != job.Edges.Release.Version {
			failed := otajob.StatusFailed
			message := fmt.Sprintf("device reported version %s after the update", reported)
			update = JobUpdate{Status: &failed, Error: &message}
		} else {
			done := 100
			update.Progress = &done
			update.ClearError = true
		}
	}
	updated, err := s.repo.UpdateJob(ctx, job.ID, update)
	if err != nil {
		return nil, fmt.Errorf("update job: %w", err)
	}
	result := toJob(updated)
	return &result, nil
}

// SummarizeOta implements device.OtaSummarizer for device listings.
func (s *Service) SummarizeOta(ctx context.Context, devices []*ent.Device) (map[uuid.UUID]device.OtaSummary, error) {
	latest, err := s.latestRelease(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(devices))
	for _, d := range devices {
		ids = append(ids, d.ID)
	}
	jobs, err := s.repo.LatestJobs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("latest jobs: %w", err)
	}
	out := make(map[uuid.UUID]device.OtaSummary, len(devices))
	for _, d := range devices {
		if summary, ok := summarize(d, latest, jobs[d.ID]); ok {
			out[d.ID] = summary
		}
	}
	return out, nil
}

func summarize(d *ent.Device, latest *ent.FirmwareRelease, job *ent.OtaJob) (device.OtaSummary, bool) {
	if job != nil && active(job.Status) && job.Edges.Release != nil {
		progress := job.Progress
		return device.OtaSummary{Status: string(job.Status), Version: job.Edges.Release.Version, Progress: &progress}, true
	}
	if d.FirmwareVersion == nil {
		return device.OtaSummary{}, false
	}
	if latest == nil || !newer(latest.Version, *d.FirmwareVersion) {
		return device.OtaSummary{Status: SummaryUpToDate}, true
	}
	if job != nil && job.Status == otajob.StatusFailed && job.Edges.Release != nil && job.Edges.Release.ID == latest.ID {
		return device.OtaSummary{Status: SummaryFailed, Version: latest.Version}, true
	}
	return device.OtaSummary{Status: SummaryUpdateAvailable, Version: latest.Version}, true
}

func (s *Service) ownedDevice(ctx context.Context, ownerID string, id uuid.UUID) (*ent.Device, error) {
	d, err := s.repo.GetOwnedDevice(ctx, ownerID, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("get device: %w", err)
	}
	return d, nil
}

// latestRelease returns the release with the highest semantic version, or nil.
func (s *Service) latestRelease(ctx context.Context) (*ent.FirmwareRelease, error) {
	releases, err := s.repo.ListReleases(ctx)
	if err != nil {
		return nil, fmt.Errorf("list releases: %w", err)
	}
	var best *ent.FirmwareRelease
	for _, release := range releases {
		if best == nil || newer(release.Version, best.Version) {
			best = release
		}
	}
	return best, nil
}

func active(status otajob.Status) bool {
	return status == otajob.StatusApproved || status == otajob.StatusDownloading || status == otajob.StatusInstalling
}

func transitionAllowed(from, to otajob.Status) bool {
	for _, allowed := range allowedTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// newer reports whether version a is a higher semantic version than b.
func newer(a, b string) bool {
	va, vb := "v"+strings.TrimPrefix(a, "v"), "v"+strings.TrimPrefix(b, "v")
	if !semver.IsValid(va) {
		return false
	}
	if !semver.IsValid(vb) {
		return true
	}
	return semver.Compare(va, vb) > 0
}

func toRelease(r *ent.FirmwareRelease) Release {
	return Release{ID: r.ID, Version: r.Version, URL: r.URL, SHA256: r.Sha256, Size: r.Size, Notes: r.Notes, CreatedAt: r.CreatedAt}
}

func toJob(j *ent.OtaJob) Job {
	job := Job{
		ID: j.ID, ReleaseID: j.ReleaseID, Status: string(j.Status), Progress: j.Progress,
		Error: j.Error, ApprovedAt: j.ApprovedAt, UpdatedAt: j.UpdatedAt,
	}
	if j.Edges.Release != nil {
		job.Version = j.Edges.Release.Version
	}
	return job
}

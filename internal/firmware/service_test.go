package firmware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/ent"
	"go-aiq-backend/internal/platform/ent/otajob"
)

type repoStub struct {
	releases []*ent.FirmwareRelease
	device   *ent.Device
	jobs     []*ent.OtaJob // newest last
	created  int
}

func (r *repoStub) CreateRelease(_ context.Context, req PublishReleaseRequest) (*ent.FirmwareRelease, error) {
	for _, existing := range r.releases {
		if existing.Version == req.Version {
			return nil, &ent.ConstraintError{}
		}
	}
	release := &ent.FirmwareRelease{ID: uuid.New(), Version: req.Version, URL: req.URL, Sha256: req.SHA256, Size: req.Size}
	r.releases = append(r.releases, release)
	return release, nil
}
func (r *repoStub) ListReleases(context.Context) ([]*ent.FirmwareRelease, error) {
	return r.releases, nil
}
func (r *repoStub) GetRelease(_ context.Context, id uuid.UUID) (*ent.FirmwareRelease, error) {
	for _, release := range r.releases {
		if release.ID == id {
			return release, nil
		}
	}
	return nil, &ent.NotFoundError{}
}
func (r *repoStub) GetOwnedDevice(_ context.Context, owner string, id uuid.UUID) (*ent.Device, error) {
	if r.device == nil || r.device.ID != id || r.device.OwnerID == nil || *r.device.OwnerID != owner {
		return nil, &ent.NotFoundError{}
	}
	return r.device, nil
}
func (r *repoStub) GetDevice(context.Context, uuid.UUID) (*ent.Device, error) { return r.device, nil }
func (r *repoStub) LatestJob(_ context.Context, deviceID uuid.UUID) (*ent.OtaJob, error) {
	for i := len(r.jobs) - 1; i >= 0; i-- {
		if r.jobs[i].DeviceID == deviceID {
			return r.jobs[i], nil
		}
	}
	return nil, nil
}
func (r *repoStub) LatestJobs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*ent.OtaJob, error) {
	out := map[uuid.UUID]*ent.OtaJob{}
	for _, id := range ids {
		if job, _ := r.LatestJob(ctx, id); job != nil {
			out[id] = job
		}
	}
	return out, nil
}
func (r *repoStub) CreateJob(ctx context.Context, deviceID, releaseID uuid.UUID, approvedBy string) (*ent.OtaJob, error) {
	release, _ := r.GetRelease(ctx, releaseID)
	job := &ent.OtaJob{ID: uuid.New(), DeviceID: deviceID, ReleaseID: releaseID, Status: otajob.StatusApproved, ApprovedBy: approvedBy, ApprovedAt: time.Now()}
	job.Edges.Release = release
	r.jobs = append(r.jobs, job)
	r.created++
	return job, nil
}
func (r *repoStub) UpdateJob(ctx context.Context, id uuid.UUID, update JobUpdate) (*ent.OtaJob, error) {
	for _, job := range r.jobs {
		if job.ID != id {
			continue
		}
		if update.Status != nil {
			job.Status = *update.Status
		}
		if update.Progress != nil {
			job.Progress = *update.Progress
		}
		if update.Error != nil {
			job.Error = update.Error
		} else if update.ClearError {
			job.Error = nil
		}
		if update.ReleaseID != nil {
			job.ReleaseID = *update.ReleaseID
			job.Edges.Release, _ = r.GetRelease(ctx, *update.ReleaseID)
		}
		return job, nil
	}
	return nil, &ent.NotFoundError{}
}

func strPtr(value string) *string { return &value }

func fixture(version string) (*repoStub, *Service) {
	owner := "user_1"
	repo := &repoStub{device: &ent.Device{ID: uuid.New(), OwnerID: &owner}}
	if version != "" {
		repo.device.FirmwareVersion = strPtr(version)
	}
	return repo, NewService(repo)
}

func publish(t *testing.T, s *Service, version string) *Release {
	t.Helper()
	release, err := s.Publish(context.Background(), PublishReleaseRequest{Version: version, URL: "https://x/fw.bin", SHA256: "ab", Size: 1000})
	if err != nil {
		t.Fatal(err)
	}
	return release
}

func TestPublishValidatesVersionSizeAndDuplicates(t *testing.T) {
	_, s := fixture("0.1.0")
	if release := publish(t, s, "v0.2.0"); release.Version != "0.2.0" {
		t.Fatalf("leading v not stripped: %q", release.Version)
	}
	ctx := context.Background()
	if _, err := s.Publish(ctx, PublishReleaseRequest{Version: "0.2.0", Size: 1}); !errors.Is(err, ErrReleaseExists) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := s.Publish(ctx, PublishReleaseRequest{Version: "latest", Size: 1}); !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("invalid: %v", err)
	}
	if _, err := s.Publish(ctx, PublishReleaseRequest{Version: "0.3.0", Size: MaxImageSize + 1}); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("too large: %v", err)
	}
}

func TestDeviceOtaOffersNewestSemverRelease(t *testing.T) {
	repo, s := fixture("0.9.0")
	publish(t, s, "0.10.0")
	publish(t, s, "0.2.0")
	state, err := s.DeviceOta(context.Background(), "user_1", repo.device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Available == nil || state.Available.Version != "0.10.0" {
		t.Fatalf("available = %+v (0.10.0 > 0.9.0 by semver, not string order)", state.Available)
	}
	if _, err := s.DeviceOta(context.Background(), "someone_else", repo.device.ID); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("other owner: %v", err)
	}
}

func TestApproveRules(t *testing.T) {
	ctx := context.Background()

	repo, s := fixture("")
	release := publish(t, s, "0.2.0")
	if _, err := s.Approve(ctx, "user_1", repo.device.ID, release.ID); !errors.Is(err, ErrFirmwareUnknown) {
		t.Fatalf("unknown firmware: %v", err)
	}

	repo, s = fixture("0.2.0")
	release = publish(t, s, "0.2.0")
	if _, err := s.Approve(ctx, "user_1", repo.device.ID, release.ID); !errors.Is(err, ErrNotNewer) {
		t.Fatalf("not newer: %v", err)
	}
	if _, err := s.Approve(ctx, "user_1", repo.device.ID, uuid.New()); !errors.Is(err, ErrReleaseNotFound) {
		t.Fatalf("missing release: %v", err)
	}

	repo, s = fixture("0.1.0")
	first := publish(t, s, "0.2.0")
	second := publish(t, s, "0.3.0")
	job, err := s.Approve(ctx, "user_1", repo.device.ID, first.ID)
	if err != nil || job.Status != "approved" {
		t.Fatalf("approve: %+v %v", job, err)
	}
	job, err = s.Approve(ctx, "user_1", repo.device.ID, second.ID)
	if err != nil || job.Version != "0.3.0" || repo.created != 1 {
		t.Fatalf("re-approve should retarget the approved job: %+v created=%d err=%v", job, repo.created, err)
	}
	repo.jobs[0].Status = otajob.StatusDownloading
	if _, err := s.Approve(ctx, "user_1", repo.device.ID, second.ID); !errors.Is(err, ErrUpdateInProgress) {
		t.Fatalf("in progress: %v", err)
	}
	if _, err := s.Cancel(ctx, "user_1", repo.device.ID); !errors.Is(err, ErrNothingToCancel) {
		t.Fatalf("cancel while downloading: %v", err)
	}
}

func TestCancelApprovedJob(t *testing.T) {
	repo, s := fixture("0.1.0")
	release := publish(t, s, "0.2.0")
	ctx := context.Background()
	if _, err := s.Approve(ctx, "user_1", repo.device.ID, release.ID); err != nil {
		t.Fatal(err)
	}
	job, err := s.Cancel(ctx, "user_1", repo.device.ID)
	if err != nil || job.Status != "cancelled" {
		t.Fatalf("cancel: %+v %v", job, err)
	}
	if pending, _ := s.DeviceCheck(ctx, repo.device.ID); pending != nil {
		t.Fatal("cancelled job must not be offered to the device")
	}
}

func TestDeviceLifecycleSucceeded(t *testing.T) {
	repo, s := fixture("0.1.0")
	release := publish(t, s, "0.2.0")
	ctx := context.Background()
	approved, _ := s.Approve(ctx, "user_1", repo.device.ID, release.ID)

	pending, err := s.DeviceCheck(ctx, repo.device.ID)
	if err != nil || pending == nil || pending.Version != "0.2.0" || pending.Status != "approved" {
		t.Fatalf("check: %+v %v", pending, err)
	}
	jobID := approved.ID.String()
	steps := []StatusReportRequest{
		{JobID: jobID, Status: "downloading", Progress: intPtr(0), Version: "0.1.0"},
		{JobID: jobID, Status: "downloading", Progress: intPtr(50), Version: "0.1.0"},
		{JobID: jobID, Status: "installing", Progress: intPtr(100), Version: "0.1.0"},
		{JobID: jobID, Status: "succeeded", Version: "0.2.0"},
	}
	for _, step := range steps {
		if _, err := s.DeviceReport(ctx, repo.device.ID, step); err != nil {
			t.Fatalf("%s: %v", step.Status, err)
		}
	}
	if repo.jobs[0].Status != otajob.StatusSucceeded || repo.jobs[0].Progress != 100 {
		t.Fatalf("final job = %+v", repo.jobs[0])
	}
	if _, err := s.DeviceReport(ctx, repo.device.ID, StatusReportRequest{JobID: jobID, Status: "downloading", Version: "0.2.0"}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("terminal job must reject reports: %v", err)
	}
}

func TestDeviceReportRollbackAndVersionMismatch(t *testing.T) {
	repo, s := fixture("0.1.0")
	release := publish(t, s, "0.2.0")
	ctx := context.Background()
	job, _ := s.Approve(ctx, "user_1", repo.device.ID, release.ID)
	repo.jobs[0].Status = otajob.StatusInstalling

	if _, err := s.DeviceReport(ctx, repo.device.ID, StatusReportRequest{JobID: uuid.NewString(), Status: "failed", Version: "0.1.0"}); !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("foreign job: %v", err)
	}
	result, err := s.DeviceReport(ctx, repo.device.ID, StatusReportRequest{JobID: job.ID.String(), Status: "succeeded", Version: "0.1.0"})
	if err != nil || result.Status != "failed" || result.Error == nil {
		t.Fatalf("success with the old version is a failure: %+v %v", result, err)
	}
}

func TestSummaries(t *testing.T) {
	latest := &ent.FirmwareRelease{ID: uuid.New(), Version: "0.2.0"}
	device := func(version string) *ent.Device {
		d := &ent.Device{ID: uuid.New()}
		if version != "" {
			d.FirmwareVersion = strPtr(version)
		}
		return d
	}
	jobFor := func(status otajob.Status) *ent.OtaJob {
		job := &ent.OtaJob{Status: status, Progress: 30}
		job.Edges.Release = latest
		return job
	}
	cases := []struct {
		name   string
		device *ent.Device
		job    *ent.OtaJob
		want   string
		ok     bool
	}{
		{"unknown firmware", device(""), nil, "", false},
		{"up to date", device("0.2.0"), nil, SummaryUpToDate, true},
		{"update available", device("0.1.0"), nil, SummaryUpdateAvailable, true},
		{"downloading", device("0.1.0"), jobFor(otajob.StatusDownloading), "downloading", true},
		{"failed", device("0.1.0"), jobFor(otajob.StatusFailed), SummaryFailed, true},
		{"cancelled shows available", device("0.1.0"), jobFor(otajob.StatusCancelled), SummaryUpdateAvailable, true},
	}
	for _, tc := range cases {
		got, ok := summarize(tc.device, latest, tc.job)
		if ok != tc.ok || got.Status != tc.want {
			t.Errorf("%s: got %+v ok=%v, want %q ok=%v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.2.0", "0.1.0", true},
		{"0.10.0", "0.9.0", true},
		{"1.0.0", "1.0.0-rc.1", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.2.0", false},
		{"bogus", "0.1.0", false},
		{"0.1.0", "bogus", true},
	}
	for _, tc := range cases {
		if got := newer(tc.a, tc.b); got != tc.want {
			t.Errorf("newer(%q, %q) = %v", tc.a, tc.b, got)
		}
	}
}

func intPtr(value int) *int { return &value }

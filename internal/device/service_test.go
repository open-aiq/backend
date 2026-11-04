package device

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go-aiq-backend/internal/platform/ent"
)

type deviceRepositoryStub struct {
	existing        *ent.Device
	updated         UpdateDeviceRequest
	byDeviceID      *ent.Device
	firmwareVersion string
	firmwareWrites  int
}

func (r *deviceRepositoryStub) Create(context.Context, string, string, string, string, bool, bool, bool) (*ent.Device, error) {
	return nil, nil
}
func (r *deviceRepositoryStub) List(context.Context, string) ([]*ent.Device, error) { return nil, nil }
func (r *deviceRepositoryStub) ListPublic(context.Context) ([]*ent.Device, error)   { return nil, nil }
func (r *deviceRepositoryStub) Get(context.Context, string, uuid.UUID) (*ent.Device, error) {
	return r.existing, nil
}
func (r *deviceRepositoryStub) GetPublic(context.Context, uuid.UUID) (*ent.Device, error) {
	return nil, nil
}
func (r *deviceRepositoryStub) GetByDeviceID(context.Context, string) (*ent.Device, error) {
	if r.byDeviceID == nil {
		return nil, &ent.NotFoundError{}
	}
	return r.byDeviceID, nil
}
func (r *deviceRepositoryStub) Update(_ context.Context, _ string, id uuid.UUID, req UpdateDeviceRequest) (*ent.Device, error) {
	r.updated = req
	locationPublic := req.IsLocationPublic != nil && *req.IsLocationPublic
	isPublic := req.IsPublic == nil || *req.IsPublic
	return &ent.Device{ID: id, Name: "Sensor", IsPublic: isPublic, IsLocationPublic: locationPublic}, nil
}
func (r *deviceRepositoryStub) UpdateKey(context.Context, string, uuid.UUID, string) (*ent.Device, error) {
	return nil, nil
}
func (r *deviceRepositoryStub) Delete(context.Context, string, uuid.UUID) error { return nil }
func (r *deviceRepositoryStub) SetFirmwareVersion(_ context.Context, _ uuid.UUID, version string) error {
	r.firmwareVersion = version
	r.firmwareWrites++
	if r.byDeviceID != nil {
		r.byDeviceID.FirmwareVersion = &version
	}
	return nil
}

func TestUpdateMakingPrivateResetsLocationConsent(t *testing.T) {
	repo := &deviceRepositoryStub{}
	value := false
	_, err := NewService(repo).Update(context.Background(), "owner", uuid.New(), UpdateDeviceRequest{IsPublic: &value})
	if err != nil {
		t.Fatal(err)
	}
	if repo.updated.IsLocationPublic == nil || *repo.updated.IsLocationPublic {
		t.Fatal("location consent was not reset")
	}
}

func TestUpdateRejectsLocationConsentForPrivateDevice(t *testing.T) {
	repo := &deviceRepositoryStub{existing: &ent.Device{IsPublic: false}}
	value := true
	_, err := NewService(repo).Update(context.Background(), "owner", uuid.New(), UpdateDeviceRequest{IsLocationPublic: &value})
	if err != ErrLocationRequiresPublic {
		t.Fatalf("got %v, want %v", err, ErrLocationRequiresPublic)
	}
}

func TestUpdateAllowsLocationConsentForExistingPublicDevice(t *testing.T) {
	repo := &deviceRepositoryStub{existing: &ent.Device{IsPublic: true}}
	value := true
	_, err := NewService(repo).Update(context.Background(), "owner", uuid.New(), UpdateDeviceRequest{IsLocationPublic: &value})
	if err != nil {
		t.Fatal(err)
	}
	if repo.updated.IsLocationPublic == nil || !*repo.updated.IsLocationPublic {
		t.Fatal("location consent was not saved")
	}
}

func TestAuthenticateDeviceRecordsChangedFirmwareVersion(t *testing.T) {
	old := "0.1.0"
	repo := &deviceRepositoryStub{byDeviceID: &ent.Device{ID: uuid.New(), DeviceKey: hashDeviceKey("sk_good"), FirmwareVersion: &old}}
	service := NewService(repo)

	id, err := service.AuthenticateDevice(context.Background(), "dev_1", "sk_good", "0.2.0")
	if err != nil || id != repo.byDeviceID.ID {
		t.Fatalf("id = %s, err = %v", id, err)
	}
	if repo.firmwareVersion != "0.2.0" || repo.firmwareWrites != 1 {
		t.Fatalf("version = %q, writes = %d", repo.firmwareVersion, repo.firmwareWrites)
	}

	if _, err := service.AuthenticateDevice(context.Background(), "dev_1", "sk_good", "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateDevice(context.Background(), "dev_1", "sk_good", ""); err != nil {
		t.Fatal(err)
	}
	if repo.firmwareWrites != 1 {
		t.Fatalf("unchanged or empty versions must not be written; writes = %d", repo.firmwareWrites)
	}
}

func TestAuthenticateDeviceRejectsBadCredentials(t *testing.T) {
	repo := &deviceRepositoryStub{byDeviceID: &ent.Device{ID: uuid.New(), DeviceKey: hashDeviceKey("sk_good")}}
	if _, err := NewService(repo).AuthenticateDevice(context.Background(), "dev_1", "sk_bad", "0.2.0"); err != ErrInvalidCredentials {
		t.Fatalf("got %v", err)
	}
	if repo.firmwareWrites != 0 {
		t.Fatal("firmware version must not be recorded for a rejected device")
	}
	if _, err := NewService(&deviceRepositoryStub{}).AuthenticateDevice(context.Background(), "dev_x", "sk", ""); err != ErrInvalidCredentials {
		t.Fatalf("unknown device: got %v", err)
	}
}

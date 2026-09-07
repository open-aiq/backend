package device

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go-aiq-backend/internal/platform/ent"
)

type deviceRepositoryStub struct {
	existing *ent.Device
	updated  UpdateDeviceRequest
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
	return nil, nil
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

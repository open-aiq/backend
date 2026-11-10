package devicelog

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/ent"
	entlog "go-aiq-backend/internal/platform/ent/devicelog"
)

type repoStub struct {
	stored []NewLog
	owned  bool
	rows   []*ent.DeviceLog
	levels []entlog.Level
	limit  int
	cutoff time.Time
}

func (r *repoStub) CreateBatch(_ context.Context, _ uuid.UUID, logs []NewLog) error {
	r.stored = append(r.stored, logs...)
	return nil
}
func (r *repoStub) DeviceOwned(context.Context, string, uuid.UUID) (bool, error) { return r.owned, nil }
func (r *repoStub) List(_ context.Context, _ uuid.UUID, levels []entlog.Level, _ int64, limit int) ([]*ent.DeviceLog, error) {
	r.levels, r.limit = levels, limit
	if len(r.rows) > limit {
		return r.rows[:limit], nil
	}
	return r.rows, nil
}
func (r *repoStub) DeleteOlderThan(_ context.Context, cutoff time.Time) (int, error) {
	r.cutoff = cutoff
	return 0, nil
}

func int64Ptr(v int64) *int64 { return &v }
func intPtr(v int) *int       { return &v }

func TestIngestStoresEntriesAndDroppedWarning(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	result, err := service.Ingest(context.Background(), uuid.New(), UploadRequest{
		Entries: []Entry{{Level: "error", Tag: "wifi", Message: strings.Repeat("é", 200), UptimeMs: int64Ptr(5), BootCount: intPtr(3), Ts: int64Ptr(1_793_700_000)}},
		Dropped: 4,
	})
	if err != nil || result.Accepted != 2 {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
	if len(repo.stored[0].Message) > messageLimit || !strings.HasPrefix(repo.stored[0].Message, "é") {
		t.Fatalf("message not truncated on a rune boundary: %d bytes", len(repo.stored[0].Message))
	}
	if repo.stored[0].DeviceTs == nil || repo.stored[0].DeviceTs.Unix() != 1_793_700_000 {
		t.Fatal("device timestamp not stored")
	}
	if repo.stored[1].Level != entlog.LevelWarn || !strings.Contains(repo.stored[1].Message, "4 entries dropped") {
		t.Fatalf("dropped warning = %+v", repo.stored[1])
	}
}

func TestListChecksOwnerFiltersLevelAndPaginates(t *testing.T) {
	rows := []*ent.DeviceLog{{ID: 9}, {ID: 8}, {ID: 7}}
	repo := &repoStub{owned: true, rows: rows}
	service := NewService(repo)
	logs, next, err := service.List(context.Background(), "u", uuid.New(), ListQuery{Level: "warn", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 || next == nil || *next != 8 {
		t.Fatalf("logs = %d, next = %v", len(logs), next)
	}
	if repo.limit != 3 {
		t.Fatalf("should fetch one extra row to detect more pages; limit = %d", repo.limit)
	}
	if len(repo.levels) != 2 || repo.levels[0] != entlog.LevelWarn {
		t.Fatalf("levels = %v", repo.levels)
	}

	repo.owned = false
	if _, _, err := service.List(context.Background(), "u", uuid.New(), ListQuery{}); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestPurgeUsesRetention(t *testing.T) {
	repo := &repoStub{}
	service := NewService(repo)
	now := time.Unix(1_800_000_000, 0)
	service.now = func() time.Time { return now }
	if _, err := service.Purge(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !repo.cutoff.Equal(now.Add(-Retention)) {
		t.Fatalf("cutoff = %s", repo.cutoff)
	}
}

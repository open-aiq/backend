package devicelog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/ent"
	entlog "go-aiq-backend/internal/platform/ent/devicelog"
)

var ErrDeviceNotFound = errors.New("device not found")

const (
	// Retention is how long logs are kept.
	Retention = 30 * 24 * time.Hour
	// UploadsPerWindow uploads are accepted per device per RateWindow; the
	// limit is enforced on the upload route (see routes.go).
	UploadsPerWindow = 20
	RateWindow       = time.Hour
	defaultLimit     = 50
	messageLimit     = 256
)

// levelOrder lists levels from least to most severe.
var levelOrder = []entlog.Level{entlog.LevelDebug, entlog.LevelInfo, entlog.LevelWarn, entlog.LevelError}

// Service handles device log ingestion and retrieval.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService creates a new device log service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Ingest stores a device's batch. A non-zero Dropped count is stored as an
// extra warning so owners know entries were lost on the device.
func (s *Service) Ingest(ctx context.Context, deviceID uuid.UUID, req UploadRequest) (*UploadResult, error) {
	logs := make([]NewLog, 0, len(req.Entries)+1)
	for _, entry := range req.Entries {
		log := NewLog{
			Level:     entlog.Level(entry.Level),
			Tag:       entry.Tag,
			Message:   truncate(entry.Message, messageLimit),
			UptimeMs:  *entry.UptimeMs,
			BootCount: *entry.BootCount,
		}
		if entry.Ts != nil {
			ts := time.Unix(*entry.Ts, 0).UTC()
			log.DeviceTs = &ts
		}
		logs = append(logs, log)
	}
	if req.Dropped > 0 {
		var uptime int64
		var boot int
		if len(logs) > 0 {
			uptime, boot = logs[0].UptimeMs, logs[0].BootCount
		}
		logs = append(logs, NewLog{
			Level: entlog.LevelWarn, Tag: "devlog", UptimeMs: uptime, BootCount: boot,
			Message: fmt.Sprintf("device log buffer overflowed; %d entries dropped", req.Dropped),
		})
	}
	if len(logs) == 0 {
		return &UploadResult{}, nil
	}
	if err := s.repo.CreateBatch(ctx, deviceID, logs); err != nil {
		return nil, fmt.Errorf("store logs: %w", err)
	}
	return &UploadResult{Accepted: len(logs)}, nil
}

// List returns a device's logs for its owner, newest first, plus the cursor
// for the next page.
func (s *Service) List(ctx context.Context, ownerID string, deviceID uuid.UUID, query ListQuery) ([]Log, *int64, error) {
	owned, err := s.repo.DeviceOwned(ctx, ownerID, deviceID)
	if err != nil {
		return nil, nil, fmt.Errorf("check owner: %w", err)
	}
	if !owned {
		return nil, nil, ErrDeviceNotFound
	}
	limit := query.Limit
	if limit == 0 {
		limit = defaultLimit
	}
	rows, err := s.repo.List(ctx, deviceID, levelsFrom(query.Level), query.Before, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list logs: %w", err)
	}
	var next *int64
	if len(rows) > limit {
		rows = rows[:limit]
		cursor := rows[len(rows)-1].ID
		next = &cursor
	}
	out := make([]Log, 0, len(rows))
	for _, row := range rows {
		out = append(out, toLog(row))
	}
	return out, next, nil
}

// Purge deletes logs older than the retention period.
func (s *Service) Purge(ctx context.Context) (int, error) {
	return s.repo.DeleteOlderThan(ctx, s.now().Add(-Retention))
}

// RunRetention purges old logs every interval until ctx is cancelled.
func (s *Service) RunRetention(ctx context.Context, interval time.Duration, onError func(error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := s.Purge(ctx); err != nil && onError != nil && ctx.Err() == nil {
			onError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func levelsFrom(minimum string) []entlog.Level {
	for i, level := range levelOrder {
		if string(level) == minimum {
			return levelOrder[i:]
		}
	}
	return levelOrder
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return strings.TrimSpace(value)
}

func toLog(row *ent.DeviceLog) Log {
	return Log{
		ID: row.ID, Level: string(row.Level), Tag: row.Tag, Message: row.Message,
		UptimeMs: row.UptimeMs, BootCount: row.BootCount, DeviceTs: row.DeviceTs, CreatedAt: row.CreatedAt,
	}
}

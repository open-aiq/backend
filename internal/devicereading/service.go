package devicereading

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service handles device reading business logic.
type Service struct {
	repo Repository
}

// NewService creates a new device reading service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Ingest stores one reading uploaded by the device with the given internal id.
func (s *Service) Ingest(ctx context.Context, deviceID uuid.UUID, req *UploadRequest) (*UploadedReading, error) {
	created, err := s.repo.Create(ctx, deviceID, req)
	if err != nil {
		return nil, fmt.Errorf("create reading: %w", err)
	}

	return &UploadedReading{
		ID:        created.ID,
		CreatedAt: created.CreatedAt,
	}, nil
}

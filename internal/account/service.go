package account

import (
	"context"
	"errors"
	"fmt"
)

// ErrMissingUser is returned when a deletion has no user to act on.
var ErrMissingUser = errors.New("missing user id")

// Service handles account business logic.
type Service struct {
	repo  Repository
	users Users
}

// NewService creates a new account service.
func NewService(repo Repository, users Users) *Service {
	return &Service{repo: repo, users: users}
}

// Delete removes everything the user owns, then the user's sign-in identity.
// The data goes first: if removing the identity fails, the user can still
// sign in and retry, whereas the reverse order could strand data that no one
// can reach.
func (s *Service) Delete(ctx context.Context, userID string) error {
	if userID == "" {
		return ErrMissingUser
	}
	if _, err := s.repo.DeleteOwnedDevices(ctx, userID); err != nil {
		return fmt.Errorf("delete owned devices: %w", err)
	}
	if err := s.users.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

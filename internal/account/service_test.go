package account

import (
	"context"
	"errors"
	"slices"
	"testing"
)

type recorder struct {
	calls     []string
	deleteErr error
	usersErr  error
}

func (r *recorder) DeleteOwnedDevices(_ context.Context, ownerID string) (int, error) {
	r.calls = append(r.calls, "devices:"+ownerID)
	return 2, r.deleteErr
}

func (r *recorder) Delete(_ context.Context, userID string) error {
	r.calls = append(r.calls, "user:"+userID)
	return r.usersErr
}

func TestDeleteRemovesDataBeforeTheUser(t *testing.T) {
	r := &recorder{}
	if err := NewService(r, r).Delete(context.Background(), "user_1"); err != nil {
		t.Fatal(err)
	}
	want := []string{"devices:user_1", "user:user_1"}
	if !slices.Equal(r.calls, want) {
		t.Fatalf("calls = %v, want %v", r.calls, want)
	}
}

func TestDeleteKeepsTheUserWhenDataDeletionFails(t *testing.T) {
	r := &recorder{deleteErr: errors.New("database down")}
	if err := NewService(r, r).Delete(context.Background(), "user_1"); err == nil {
		t.Fatal("expected an error")
	}
	if slices.Contains(r.calls, "user:user_1") {
		t.Fatalf("user was deleted even though its data wasn't: %v", r.calls)
	}
}

func TestDeleteReportsIdentityFailures(t *testing.T) {
	r := &recorder{usersErr: errors.New("clerk down")}
	if err := NewService(r, r).Delete(context.Background(), "user_1"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestDeleteNeedsAUser(t *testing.T) {
	r := &recorder{}
	if err := NewService(r, r).Delete(context.Background(), ""); !errors.Is(err, ErrMissingUser) {
		t.Fatalf("err = %v, want ErrMissingUser", err)
	}
	if len(r.calls) != 0 {
		t.Fatalf("unexpected calls: %v", r.calls)
	}
}

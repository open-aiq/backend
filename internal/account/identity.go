package account

import (
	"context"
	"errors"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
)

// Users removes sign-in identities from the identity provider.
type Users interface {
	// Delete removes the user. Deleting a user that no longer exists succeeds,
	// so a failed account deletion can be retried.
	Delete(ctx context.Context, userID string) error
}

// ClerkUsers deletes users through the Clerk Backend API, using the secret
// key set with clerk.SetKey.
type ClerkUsers struct{}

// Delete removes the Clerk user, treating "not found" as already done.
func (ClerkUsers) Delete(ctx context.Context, userID string) error {
	_, err := user.Delete(ctx, userID)
	var apiErr *clerk.APIErrorResponse
	if errors.As(err, &apiErr) && apiErr.HTTPStatusCode == http.StatusNotFound {
		return nil
	}
	return err
}

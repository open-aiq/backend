package account

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"go-aiq-backend/internal/platform/input"
	"go-aiq-backend/internal/platform/middleware"
	"go-aiq-backend/internal/platform/problem"
)

// Handler handles HTTP requests for the signed-in user's account.
type Handler struct {
	service *Service
}

// NewHandler creates a new account handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Delete godoc
//
// @Summary Delete your account
// @Description Permanently deletes every device you own, with its readings, logs and firmware update history, then your sign-in account. If it fails partway, calling it again finishes the job.
// @Tags Account
// @Produce json
//
// @Success 204 "No Content"
// @Failure 401 {object} problem.Problem
// @Failure 422 {object} problem.Problem
// @Failure 500 {object} problem.Problem
//
// @Security BearerAuth
// @Router /account [delete]
func (h *Handler) Delete(c *gin.Context) {
	if !input.RejectUnknownQuery(c) {
		return
	}

	if err := h.service.Delete(c.Request.Context(), middleware.UserID(c)); err != nil {
		if errors.Is(err, ErrMissingUser) {
			problem.Write(c, problem.Unauthorized, "Sign in to delete your account.")
			return
		}
		_ = c.Error(err)
		problem.Write(c, problem.Internal, "Failed to delete the account.")
		return
	}

	c.Status(http.StatusNoContent)
}

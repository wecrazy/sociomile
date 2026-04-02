package service

import (
	"context"

	"github.com/wecrazy/sociomile/backend/internal/model"
	"github.com/wecrazy/sociomile/backend/internal/repository"
)

// UserService handles tenant-scoped user lookups.
type UserService struct {
	store *repository.Store
}

// NewUserService builds a UserService.
func NewUserService(store *repository.Store) *UserService {
	return &UserService{store: store}
}

// ListAgents returns the active agents for the given tenant.
func (s *UserService) ListAgents(ctx context.Context, tenantID string) ([]model.User, error) {
	return s.store.ListUsersByRole(ctx, tenantID, model.RoleAgent)
}

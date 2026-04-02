package service

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/wecrazy/sociomile/backend/internal/apperror"
	"github.com/wecrazy/sociomile/backend/internal/auth"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"github.com/wecrazy/sociomile/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles login and self-profile lookups.
type AuthService struct {
	store  *repository.Store
	secret string
	ttl    time.Duration
}

// LoginResult contains the issued access token and the authenticated user.
type LoginResult struct {
	AccessToken string      `json:"access_token"`
	User        *model.User `json:"user"`
}

// NewAuthService builds an AuthService.
func NewAuthService(store *repository.Store, secret string, ttl time.Duration) *AuthService {
	return &AuthService{store: store, secret: secret, ttl: ttl}
}

// Login authenticates a user and returns a signed access token.
func (s *AuthService) Login(ctx context.Context, email string, password string) (*LoginResult, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, apperror.New(fiber.StatusUnauthorized, "invalid_credentials", "email or password is invalid")
	}

	if !user.Active {
		return nil, apperror.New(fiber.StatusForbidden, "inactive_user", "user is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, apperror.New(fiber.StatusUnauthorized, "invalid_credentials", "email or password is invalid")
	}

	token, err := auth.GenerateToken(s.secret, s.ttl, user)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken: token,
		User:        user,
	}, nil
}

// GetMe loads the authenticated user for the current tenant.
func (s *AuthService) GetMe(ctx context.Context, userID string, tenantID string) (*model.User, error) {
	user, err := s.store.GetUserByIDAndTenant(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

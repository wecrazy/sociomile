package handler

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/wecrazy/sociomile/backend/internal/apperror"
	"github.com/wecrazy/sociomile/backend/internal/http/middleware"
	"github.com/wecrazy/sociomile/backend/internal/http/response"
	"github.com/wecrazy/sociomile/backend/internal/service"
)

// AuthHandler serves authentication-related API endpoints.
type AuthHandler struct {
	service *service.AuthService
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// Login authenticates a user and returns a JWT plus profile data.
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var request loginRequest
	if err := c.Bind().Body(&request); err != nil {
		return response.Error(c, apperror.New(fiber.StatusBadRequest, "invalid_request", "request body must be valid JSON"))
	}

	request.Email = strings.TrimSpace(request.Email)
	if request.Email == "" || request.Password == "" {
		return response.Error(c, apperror.New(fiber.StatusBadRequest, "invalid_request", "email and password are required"))
	}

	result, err := h.service.Login(c.Context(), request.Email, request.Password)
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusOK, result)
}

// Me returns the authenticated user profile.
func (h *AuthHandler) Me(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	user, err := h.service.GetMe(c.Context(), claims.UserID, claims.TenantID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusOK, user)
}

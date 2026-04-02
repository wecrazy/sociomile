package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/wecrazy/sociomile/backend/internal/apperror"
	"github.com/wecrazy/sociomile/backend/internal/http/middleware"
	"github.com/wecrazy/sociomile/backend/internal/http/response"
	"github.com/wecrazy/sociomile/backend/internal/service"
)

// UserHandler serves user-related API endpoints.
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler builds a UserHandler.
func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// ListAgents returns the active agents for the current tenant.
func (h *UserHandler) ListAgents(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	users, err := h.service.ListAgents(c.Context(), claims.TenantID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusOK, users)
}

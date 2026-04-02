package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/wecrazy/sociomile/backend/internal/apperror"
	"github.com/wecrazy/sociomile/backend/internal/http/middleware"
	"github.com/wecrazy/sociomile/backend/internal/http/response"
	"github.com/wecrazy/sociomile/backend/internal/service"
)

// TicketHandler serves ticket-related API endpoints.
type TicketHandler struct {
	service *service.TicketService
}

// NewTicketHandler builds a TicketHandler.
func NewTicketHandler(service *service.TicketService) *TicketHandler {
	return &TicketHandler{service: service}
}

// List returns the paginated ticket queue for the current tenant.
func (h *TicketHandler) List(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	offset, limit := parsePagination(c)
	tickets, total, err := h.service.ListTickets(c.Context(), claims.TenantID, service.ListTicketsInput{
		Status:          c.Query("status"),
		Priority:        c.Query("priority"),
		AssignedAgentID: c.Query("assigned_agent_id"),
		Offset:          offset,
		Limit:           limit,
	})
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, tickets, total, offset, limit, map[string]any{
		"status":            c.Query("status"),
		"priority":          c.Query("priority"),
		"assigned_agent_id": c.Query("assigned_agent_id"),
	})
}

// Detail returns one ticket for the current tenant.
func (h *TicketHandler) Detail(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	ticket, err := h.service.GetTicket(c.Context(), claims.TenantID, c.Params("id"))
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusOK, ticket)
}

// Escalate converts a conversation into a ticket.
func (h *TicketHandler) Escalate(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	var input service.EscalateTicketInput
	if err := c.Bind().Body(&input); err != nil {
		return response.Error(c, apperror.New(fiber.StatusBadRequest, "invalid_request", "request body must be valid JSON"))
	}

	ticket, err := h.service.EscalateConversation(c.Context(), claims.TenantID, c.Params("id"), claims.UserID, input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusCreated, ticket)
}

// UpdateStatus changes a ticket status for the current tenant.
func (h *TicketHandler) UpdateStatus(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	var input service.UpdateTicketStatusInput
	if err := c.Bind().Body(&input); err != nil {
		return response.Error(c, apperror.New(fiber.StatusBadRequest, "invalid_request", "request body must be valid JSON"))
	}

	ticket, err := h.service.UpdateStatus(c.Context(), claims.TenantID, c.Params("id"), claims.UserID, claims.Role, input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusOK, ticket)
}

package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/wecrazy/sociomile/backend/internal/apperror"
	"github.com/wecrazy/sociomile/backend/internal/http/middleware"
	"github.com/wecrazy/sociomile/backend/internal/http/response"
	"github.com/wecrazy/sociomile/backend/internal/service"
)

// ConversationHandler serves conversation-related API endpoints.
type ConversationHandler struct {
	service *service.ConversationService
}

// NewConversationHandler builds a ConversationHandler.
func NewConversationHandler(service *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{service: service}
}

// ChannelWebhook receives inbound channel events and updates conversations.
func (h *ConversationHandler) ChannelWebhook(c fiber.Ctx) error {
	var input service.WebhookInput
	if err := c.Bind().Body(&input); err != nil {
		return response.Error(c, apperror.New(fiber.StatusBadRequest, "invalid_request", "request body must be valid JSON"))
	}

	if err := h.service.AllowWebhook(c.Context(), input.TenantID, c.IP()); err != nil {
		return response.Error(c, err)
	}

	conversation, err := h.service.HandleWebhook(c.Context(), input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusCreated, conversation)
}

// List returns the paginated conversation queue for the current tenant.
func (h *ConversationHandler) List(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	offset, limit := parsePagination(c)
	conversations, total, err := h.service.ListConversations(c.Context(), claims.TenantID, service.ListConversationsInput{
		Status:          c.Query("status"),
		AssignedAgentID: c.Query("assigned_agent_id"),
		Offset:          offset,
		Limit:           limit,
	})
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, conversations, total, offset, limit, map[string]any{
		"status":            c.Query("status"),
		"assigned_agent_id": c.Query("assigned_agent_id"),
	})
}

// Detail returns one conversation for the current tenant.
func (h *ConversationHandler) Detail(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	conversation, err := h.service.GetConversation(c.Context(), claims.TenantID, c.Params("id"))
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusOK, conversation)
}

// Assign assigns a conversation to an agent.
func (h *ConversationHandler) Assign(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	var input service.AssignConversationInput
	if err := c.Bind().Body(&input); err != nil {
		return response.Error(c, apperror.New(fiber.StatusBadRequest, "invalid_request", "request body must be valid JSON"))
	}

	conversation, err := h.service.AssignConversation(c.Context(), claims.TenantID, c.Params("id"), claims.UserID, claims.Role, input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusOK, conversation)
}

// Reply posts an agent reply to a conversation.
func (h *ConversationHandler) Reply(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	var input service.ReplyConversationInput
	if err := c.Bind().Body(&input); err != nil {
		return response.Error(c, apperror.New(fiber.StatusBadRequest, "invalid_request", "request body must be valid JSON"))
	}

	conversation, err := h.service.ReplyToConversation(c.Context(), claims.TenantID, c.Params("id"), claims.UserID, input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusOK, conversation)
}

// Close closes a conversation for the current tenant.
func (h *ConversationHandler) Close(c fiber.Ctx) error {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		return response.Error(c, apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing"))
	}

	conversation, err := h.service.CloseConversation(c.Context(), claims.TenantID, c.Params("id"), claims.UserID, claims.Role)
	if err != nil {
		return response.Error(c, err)
	}

	return response.JSON(c, fiber.StatusOK, conversation)
}

func parsePagination(c fiber.Ctx) (int, int) {
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil {
		offset = 0
	}

	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil {
		limit = 20
	}

	return offset, limit
}

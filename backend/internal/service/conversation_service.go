package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/wecrazy/sociomile/backend/internal/apperror"
	"github.com/wecrazy/sociomile/backend/internal/cache"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"github.com/wecrazy/sociomile/backend/internal/repository"
	"gorm.io/gorm"
)

// ConversationService handles conversation intake and lifecycle operations.
type ConversationService struct {
	store *repository.Store
	cache *cache.Client
}

// WebhookInput contains the inbound channel message payload.
type WebhookInput struct {
	TenantID           string `json:"tenant_id"`
	ChannelKey         string `json:"channel_key"`
	CustomerExternalID string `json:"customer_external_id"`
	CustomerName       string `json:"customer_name"`
	Message            string `json:"message"`
}

// ListConversationsInput contains the server-side conversation list filters.
type ListConversationsInput struct {
	Status          string
	AssignedAgentID string
	Offset          int
	Limit           int
}

// AssignConversationInput contains the target agent identifier.
type AssignConversationInput struct {
	AgentID string `json:"agent_id"`
}

// ReplyConversationInput contains the outbound reply text.
type ReplyConversationInput struct {
	Message string `json:"message"`
}

type conversationListCache struct {
	Data  []model.Conversation `json:"data"`
	Total int64                `json:"total"`
}

// NewConversationService builds a ConversationService.
func NewConversationService(store *repository.Store, cacheClient *cache.Client) *ConversationService {
	return &ConversationService{store: store, cache: cacheClient}
}

// AllowWebhook applies tenant-scoped rate limiting to inbound webhook traffic.
func (s *ConversationService) AllowWebhook(ctx context.Context, tenantID string, ipAddress string) error {
	if tenantID == "" {
		return apperror.New(fiber.StatusBadRequest, "invalid_tenant", "tenant_id is required")
	}

	key := s.cache.Key("tenant", tenantID, "webhook", "rate", ipAddress)
	allowed, err := s.cache.CheckRateLimit(ctx, key, 30, time.Minute)
	if err != nil {
		return nil
	}
	if !allowed {
		return apperror.New(fiber.StatusTooManyRequests, "rate_limited", "webhook rate limit exceeded")
	}

	return nil
}

// HandleWebhook creates or updates a conversation from an inbound channel message.
func (s *ConversationService) HandleWebhook(ctx context.Context, input WebhookInput) (*model.Conversation, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.ChannelKey = strings.TrimSpace(input.ChannelKey)
	input.CustomerExternalID = strings.TrimSpace(input.CustomerExternalID)
	input.CustomerName = strings.TrimSpace(input.CustomerName)
	input.Message = strings.TrimSpace(input.Message)

	if input.TenantID == "" || input.ChannelKey == "" || input.CustomerExternalID == "" || input.Message == "" {
		return nil, apperror.New(fiber.StatusBadRequest, "invalid_webhook", "tenant_id, channel_key, customer_external_id, and message are required")
	}

	var conversationID string
	createdConversation := false

	err := s.store.WithinTransaction(ctx, func(tx *repository.Store) error {
		if _, err := tx.GetTenant(ctx, input.TenantID); err != nil {
			return apperror.New(fiber.StatusNotFound, "tenant_not_found", "tenant not found")
		}

		channel, err := tx.GetChannelByKey(ctx, input.TenantID, input.ChannelKey)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.New(fiber.StatusBadRequest, "invalid_channel", "channel_key is not configured for the tenant")
			}

			return err
		}

		customer, err := tx.GetOrCreateCustomer(ctx, input.TenantID, input.CustomerExternalID, input.CustomerName)
		if err != nil {
			return err
		}

		conversation, err := tx.FindActiveConversation(ctx, input.TenantID, customer.ID, channel.ID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			conversation = &model.Conversation{
				TenantID:   input.TenantID,
				CustomerID: customer.ID,
				ChannelID:  channel.ID,
				Status:     model.ConversationStatusOpen,
			}
			if err := tx.CreateConversation(ctx, conversation); err != nil {
				return err
			}
			createdConversation = true
		}

		message := &model.Message{
			ConversationID: conversation.ID,
			SenderType:     model.SenderTypeCustomer,
			Message:        input.Message,
		}
		if err := tx.CreateMessage(ctx, message); err != nil {
			return err
		}

		conversation.UpdatedAt = time.Now()
		if err := tx.SaveConversation(ctx, conversation); err != nil {
			return err
		}

		if createdConversation {
			if err := appendDomainEvent(ctx, tx, input.TenantID, "conversation.created", "conversation", conversation.ID, conversation); err != nil {
				return err
			}
		}

		if err := appendDomainEvent(ctx, tx, input.TenantID, "channel.message.received", "conversation", conversation.ID, message); err != nil {
			return err
		}

		conversationID = conversation.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.cache.BumpVersion(ctx, input.TenantID, "conversations")
	return s.GetConversation(ctx, input.TenantID, conversationID)
}

// ListConversations returns a paginated conversation list for the given tenant.
func (s *ConversationService) ListConversations(ctx context.Context, tenantID string, input ListConversationsInput) ([]model.Conversation, int64, error) {
	filter := repository.ConversationFilter{
		Status:          strings.TrimSpace(input.Status),
		AssignedAgentID: strings.TrimSpace(input.AssignedAgentID),
		Offset:          normalizeOffset(input.Offset),
		Limit:           normalizeLimit(input.Limit),
	}

	version := s.cache.Version(ctx, tenantID, "conversations")
	cacheKey := s.cache.Key(
		"tenant", tenantID,
		"conversations",
		fmt.Sprintf("v%d", version),
		fmt.Sprintf("offset=%d", filter.Offset),
		fmt.Sprintf("limit=%d", filter.Limit),
		fmt.Sprintf("status=%s", filter.Status),
		fmt.Sprintf("agent=%s", filter.AssignedAgentID),
	)

	var cached conversationListCache
	if hit, err := s.cache.GetJSON(ctx, cacheKey, &cached); err == nil && hit {
		return cached.Data, cached.Total, nil
	}

	conversations, total, err := s.store.ListConversations(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}

	_ = s.cache.SetJSON(ctx, cacheKey, conversationListCache{Data: conversations, Total: total}, 5*time.Minute)
	return conversations, total, nil
}

// GetConversation returns one conversation for the given tenant.
func (s *ConversationService) GetConversation(ctx context.Context, tenantID string, conversationID string) (*model.Conversation, error) {
	return s.store.GetConversationByID(ctx, tenantID, conversationID)
}

// AssignConversation assigns a conversation to an agent.
func (s *ConversationService) AssignConversation(ctx context.Context, tenantID string, conversationID string, actorID string, actorRole string, input AssignConversationInput) (*model.Conversation, error) {
	if actorRole != model.RoleAdmin {
		return nil, apperror.New(fiber.StatusForbidden, "forbidden", "only admins can assign conversations")
	}
	if _, err := s.store.GetUserByIDAndTenant(ctx, actorID, tenantID); err != nil {
		return nil, apperror.New(fiber.StatusForbidden, "forbidden", "admin user is not available in this tenant")
	}

	agentID := strings.TrimSpace(input.AgentID)
	if agentID == "" {
		return nil, apperror.New(fiber.StatusBadRequest, "invalid_assignment", "agent_id is required")
	}

	agent, err := s.store.GetUserByIDAndTenant(ctx, agentID, tenantID)
	if err != nil {
		return nil, apperror.New(fiber.StatusBadRequest, "invalid_assignment", "agent does not exist in this tenant")
	}
	if agent.Role != model.RoleAgent {
		return nil, apperror.New(fiber.StatusBadRequest, "invalid_assignment", "target user must have the agent role")
	}

	if err := s.store.WithinTransaction(ctx, func(tx *repository.Store) error {
		conversation, err := tx.GetConversationByID(ctx, tenantID, conversationID)
		if err != nil {
			return err
		}
		if conversation.Status == model.ConversationStatusClosed {
			return apperror.New(fiber.StatusConflict, "conversation_closed", "closed conversations cannot be assigned")
		}

		conversation.AssignedAgentID = &agent.ID
		conversation.Status = model.ConversationStatusAssigned
		if err := tx.SaveConversation(ctx, conversation); err != nil {
			return err
		}

		return appendDomainEvent(ctx, tx, tenantID, "conversation.assigned", "conversation", conversation.ID, map[string]any{
			"conversation_id": conversation.ID,
			"agent_id":        agent.ID,
		})
	}); err != nil {
		return nil, err
	}

	s.cache.BumpVersion(ctx, tenantID, "conversations")
	return s.GetConversation(ctx, tenantID, conversationID)
}

// ReplyToConversation posts an agent reply to a conversation.
func (s *ConversationService) ReplyToConversation(ctx context.Context, tenantID string, conversationID string, agentID string, input ReplyConversationInput) (*model.Conversation, error) {
	messageText := strings.TrimSpace(input.Message)
	if messageText == "" {
		return nil, apperror.New(fiber.StatusBadRequest, "invalid_message", "message is required")
	}

	agent, err := s.store.GetUserByIDAndTenant(ctx, agentID, tenantID)
	if err != nil || agent.Role != model.RoleAgent {
		return nil, apperror.New(fiber.StatusForbidden, "forbidden", "only agents can reply to conversations")
	}

	if err := s.store.WithinTransaction(ctx, func(tx *repository.Store) error {
		conversation, err := tx.GetConversationByID(ctx, tenantID, conversationID)
		if err != nil {
			return err
		}
		if conversation.Status == model.ConversationStatusClosed {
			return apperror.New(fiber.StatusConflict, "conversation_closed", "cannot reply to a closed conversation")
		}
		if conversation.AssignedAgentID != nil && *conversation.AssignedAgentID != agent.ID {
			return apperror.New(fiber.StatusForbidden, "conversation_assigned_elsewhere", "conversation is assigned to a different agent")
		}

		if conversation.AssignedAgentID == nil {
			conversation.AssignedAgentID = &agent.ID
			conversation.Status = model.ConversationStatusAssigned
		}

		message := &model.Message{
			ConversationID: conversation.ID,
			SenderType:     model.SenderTypeAgent,
			SenderID:       &agent.ID,
			Message:        messageText,
		}
		if err := tx.CreateMessage(ctx, message); err != nil {
			return err
		}

		conversation.UpdatedAt = time.Now()
		if err := tx.SaveConversation(ctx, conversation); err != nil {
			return err
		}

		return appendDomainEvent(ctx, tx, tenantID, "conversation.replied", "conversation", conversation.ID, map[string]any{
			"conversation_id": conversation.ID,
			"agent_id":        agent.ID,
			"message_id":      message.ID,
		})
	}); err != nil {
		return nil, err
	}

	s.cache.BumpVersion(ctx, tenantID, "conversations")
	return s.GetConversation(ctx, tenantID, conversationID)
}

// CloseConversation closes a conversation for the current tenant.
func (s *ConversationService) CloseConversation(ctx context.Context, tenantID string, conversationID string, actorID string, actorRole string) (*model.Conversation, error) {
	if err := s.store.WithinTransaction(ctx, func(tx *repository.Store) error {
		conversation, err := tx.GetConversationByID(ctx, tenantID, conversationID)
		if err != nil {
			return err
		}

		if actorRole == model.RoleAgent {
			if conversation.AssignedAgentID == nil || *conversation.AssignedAgentID != actorID {
				return apperror.New(fiber.StatusForbidden, "forbidden", "agent can only close conversations assigned to them")
			}
		}

		if conversation.Status == model.ConversationStatusClosed {
			return nil
		}

		conversation.Status = model.ConversationStatusClosed
		if err := tx.SaveConversation(ctx, conversation); err != nil {
			return err
		}

		return appendDomainEvent(ctx, tx, tenantID, "conversation.closed", "conversation", conversation.ID, map[string]any{
			"conversation_id": conversation.ID,
			"actor_id":        actorID,
			"actor_role":      actorRole,
		})
	}); err != nil {
		return nil, err
	}

	s.cache.BumpVersion(ctx, tenantID, "conversations")
	return s.GetConversation(ctx, tenantID, conversationID)
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}

	return offset
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}

	return limit
}

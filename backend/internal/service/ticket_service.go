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

// TicketService handles ticket lifecycle operations.
type TicketService struct {
	store *repository.Store
	cache *cache.Client
}

// ListTicketsInput contains the server-side ticket list filters.
type ListTicketsInput struct {
	Status          string
	Priority        string
	AssignedAgentID string
	Offset          int
	Limit           int
}

// EscalateTicketInput contains the data needed to create a ticket from a conversation.
type EscalateTicketInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

// UpdateTicketStatusInput contains the next ticket status.
type UpdateTicketStatusInput struct {
	Status string `json:"status"`
}

type ticketListCache struct {
	Data  []model.Ticket `json:"data"`
	Total int64          `json:"total"`
}

// NewTicketService builds a TicketService.
func NewTicketService(store *repository.Store, cacheClient *cache.Client) *TicketService {
	return &TicketService{store: store, cache: cacheClient}
}

// ListTickets returns a paginated ticket list for the given tenant.
func (s *TicketService) ListTickets(ctx context.Context, tenantID string, input ListTicketsInput) ([]model.Ticket, int64, error) {
	filter := repository.TicketFilter{
		Status:          strings.TrimSpace(input.Status),
		Priority:        strings.TrimSpace(input.Priority),
		AssignedAgentID: strings.TrimSpace(input.AssignedAgentID),
		Offset:          normalizeOffset(input.Offset),
		Limit:           normalizeLimit(input.Limit),
	}

	version := s.cache.Version(ctx, tenantID, "tickets")
	cacheKey := s.cache.Key(
		"tenant", tenantID,
		"tickets",
		fmt.Sprintf("v%d", version),
		fmt.Sprintf("offset=%d", filter.Offset),
		fmt.Sprintf("limit=%d", filter.Limit),
		fmt.Sprintf("status=%s", filter.Status),
		fmt.Sprintf("priority=%s", filter.Priority),
		fmt.Sprintf("agent=%s", filter.AssignedAgentID),
	)

	var cached ticketListCache
	if hit, err := s.cache.GetJSON(ctx, cacheKey, &cached); err == nil && hit {
		return cached.Data, cached.Total, nil
	}

	tickets, total, err := s.store.ListTickets(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}

	_ = s.cache.SetJSON(ctx, cacheKey, ticketListCache{Data: tickets, Total: total}, 5*time.Minute)
	return tickets, total, nil
}

// GetTicket returns one ticket for the given tenant.
func (s *TicketService) GetTicket(ctx context.Context, tenantID string, ticketID string) (*model.Ticket, error) {
	return s.store.GetTicketByID(ctx, tenantID, ticketID)
}

// EscalateConversation creates a ticket from a conversation.
func (s *TicketService) EscalateConversation(ctx context.Context, tenantID string, conversationID string, agentID string, input EscalateTicketInput) (*model.Ticket, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Priority = strings.TrimSpace(input.Priority)
	if input.Title == "" || input.Description == "" {
		return nil, apperror.New(fiber.StatusBadRequest, "invalid_ticket", "title and description are required")
	}
	if input.Priority == "" {
		input.Priority = model.TicketPriorityMedium
	}
	if !isValidPriority(input.Priority) {
		return nil, apperror.New(fiber.StatusBadRequest, "invalid_priority", "priority must be low, medium, or high")
	}

	agent, err := s.store.GetUserByIDAndTenant(ctx, agentID, tenantID)
	if err != nil || agent.Role != model.RoleAgent {
		return nil, apperror.New(fiber.StatusForbidden, "forbidden", "only agents can escalate conversations")
	}

	var ticketID string
	err = s.store.WithinTransaction(ctx, func(tx *repository.Store) error {
		conversation, err := tx.GetConversationByID(ctx, tenantID, conversationID)
		if err != nil {
			return err
		}
		if conversation.Status == model.ConversationStatusClosed {
			return apperror.New(fiber.StatusConflict, "conversation_closed", "cannot escalate a closed conversation")
		}
		if conversation.AssignedAgentID != nil && *conversation.AssignedAgentID != agent.ID {
			return apperror.New(fiber.StatusForbidden, "conversation_assigned_elsewhere", "conversation is assigned to a different agent")
		}

		if _, err := tx.FindTicketByConversation(ctx, tenantID, conversationID); err == nil {
			return apperror.New(fiber.StatusConflict, "ticket_exists", "conversation already has a ticket")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if conversation.AssignedAgentID == nil {
			conversation.AssignedAgentID = &agent.ID
			conversation.Status = model.ConversationStatusAssigned
			if err := tx.SaveConversation(ctx, conversation); err != nil {
				return err
			}
		}

		ticket := &model.Ticket{
			TenantID:        tenantID,
			ConversationID:  conversation.ID,
			Title:           input.Title,
			Description:     input.Description,
			Status:          model.TicketStatusOpen,
			Priority:        input.Priority,
			AssignedAgentID: conversation.AssignedAgentID,
		}
		if err := tx.CreateTicket(ctx, ticket); err != nil {
			return err
		}

		if err := appendDomainEvent(ctx, tx, tenantID, "conversation.escalated", "conversation", conversation.ID, map[string]any{
			"conversation_id": conversation.ID,
			"ticket_id":       ticket.ID,
			"agent_id":        agent.ID,
		}); err != nil {
			return err
		}

		if err := appendDomainEvent(ctx, tx, tenantID, "ticket.created", "ticket", ticket.ID, ticket); err != nil {
			return err
		}

		ticketID = ticket.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.cache.BumpVersion(ctx, tenantID, "conversations")
	s.cache.BumpVersion(ctx, tenantID, "tickets")
	return s.GetTicket(ctx, tenantID, ticketID)
}

// UpdateStatus changes the status of a ticket.
func (s *TicketService) UpdateStatus(ctx context.Context, tenantID string, ticketID string, actorID string, actorRole string, input UpdateTicketStatusInput) (*model.Ticket, error) {
	if actorRole != model.RoleAdmin {
		return nil, apperror.New(fiber.StatusForbidden, "forbidden", "only admins can update ticket status")
	}
	if _, err := s.store.GetUserByIDAndTenant(ctx, actorID, tenantID); err != nil {
		return nil, apperror.New(fiber.StatusForbidden, "forbidden", "admin user is not available in this tenant")
	}

	status := strings.TrimSpace(input.Status)
	if !isValidTicketStatus(status) {
		return nil, apperror.New(fiber.StatusBadRequest, "invalid_status", "invalid ticket status")
	}

	if err := s.store.WithinTransaction(ctx, func(tx *repository.Store) error {
		ticket, err := tx.GetTicketByID(ctx, tenantID, ticketID)
		if err != nil {
			return err
		}

		ticket.Status = status
		if err := tx.SaveTicket(ctx, ticket); err != nil {
			return err
		}

		return appendDomainEvent(ctx, tx, tenantID, "ticket.status.updated", "ticket", ticket.ID, map[string]any{
			"ticket_id": ticket.ID,
			"status":    ticket.Status,
		})
	}); err != nil {
		return nil, err
	}

	s.cache.BumpVersion(ctx, tenantID, "tickets")
	return s.GetTicket(ctx, tenantID, ticketID)
}

func isValidPriority(priority string) bool {
	switch priority {
	case model.TicketPriorityLow, model.TicketPriorityMedium, model.TicketPriorityHigh:
		return true
	default:
		return false
	}
}

func isValidTicketStatus(status string) bool {
	switch status {
	case model.TicketStatusOpen, model.TicketStatusInProgress, model.TicketStatusResolved, model.TicketStatusClosed:
		return true
	default:
		return false
	}
}

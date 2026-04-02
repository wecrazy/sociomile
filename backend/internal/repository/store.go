package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wecrazy/sociomile/backend/internal/model"
	"gorm.io/gorm"
)

// ConversationFilter contains tenant-scoped conversation list filters.
type ConversationFilter struct {
	Status          string
	AssignedAgentID string
	Offset          int
	Limit           int
}

// TicketFilter contains tenant-scoped ticket list filters.
type TicketFilter struct {
	Status          string
	Priority        string
	AssignedAgentID string
	Offset          int
	Limit           int
}

// Store wraps tenant-aware persistence helpers for the application.
type Store struct {
	db *gorm.DB
}

// NewStore builds a repository store around a GORM database handle.
func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// DB returns the underlying GORM database handle.
func (s *Store) DB() *gorm.DB {
	return s.db
}

// WithinTransaction runs fn inside a database transaction.
func (s *Store) WithinTransaction(ctx context.Context, fn func(tx *Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Store{db: tx})
	})
}

// GetTenant loads one tenant by identifier.
func (s *Store) GetTenant(ctx context.Context, tenantID string) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := s.db.WithContext(ctx).Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		return nil, err
	}

	return &tenant, nil
}

// GetUserByEmail loads one user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Preload("Tenant").Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByIDAndTenant loads one user by identifier within a tenant.
func (s *Store) GetUserByIDAndTenant(ctx context.Context, userID string, tenantID string) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// ListUsersByRole returns active users for a tenant and role.
func (s *Store) ListUsersByRole(ctx context.Context, tenantID string, role string) ([]model.User, error) {
	var users []model.User
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND role = ? AND active = ?", tenantID, role, true).
		Order("name ASC").
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

// GetChannelByKey loads one channel by tenant and channel key.
func (s *Store) GetChannelByKey(ctx context.Context, tenantID string, key string) (*model.Channel, error) {
	var channel model.Channel
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND `key` = ?", tenantID, key).First(&channel).Error; err != nil {
		return nil, err
	}

	return &channel, nil
}

// GetOrCreateCustomer loads or creates a tenant-scoped customer.
func (s *Store) GetOrCreateCustomer(ctx context.Context, tenantID string, externalID string, name string) (*model.Customer, error) {
	var customer model.Customer
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND external_id = ?", tenantID, externalID).First(&customer).Error
	if err == nil {
		if name != "" && customer.Name != name {
			customer.Name = name
			if saveErr := s.db.WithContext(ctx).Save(&customer).Error; saveErr != nil {
				return nil, saveErr
			}
		}

		return &customer, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if name == "" {
		name = externalID
	}

	customer = model.Customer{
		TenantID:   tenantID,
		ExternalID: externalID,
		Name:       name,
	}

	if err := s.db.WithContext(ctx).Create(&customer).Error; err != nil {
		return nil, err
	}

	return &customer, nil
}

// FindActiveConversation loads the newest open or assigned conversation for a customer and channel.
func (s *Store) FindActiveConversation(ctx context.Context, tenantID string, customerID string, channelID string) (*model.Conversation, error) {
	var conversation model.Conversation
	err := s.db.WithContext(ctx).
		Preload("Customer").
		Preload("Channel").
		Preload("AssignedAgent").
		Where("tenant_id = ? AND customer_id = ? AND channel_id = ? AND status IN ?", tenantID, customerID, channelID, []string{model.ConversationStatusOpen, model.ConversationStatusAssigned}).
		Order("updated_at DESC").
		First(&conversation).Error
	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

// CreateConversation inserts a new conversation.
func (s *Store) CreateConversation(ctx context.Context, conversation *model.Conversation) error {
	return s.db.WithContext(ctx).Create(conversation).Error
}

// SaveConversation updates an existing conversation.
func (s *Store) SaveConversation(ctx context.Context, conversation *model.Conversation) error {
	return s.db.WithContext(ctx).Save(conversation).Error
}

// CreateMessage inserts a new message.
func (s *Store) CreateMessage(ctx context.Context, message *model.Message) error {
	return s.db.WithContext(ctx).Create(message).Error
}

// GetConversationByID loads one conversation with related records.
func (s *Store) GetConversationByID(ctx context.Context, tenantID string, conversationID string) (*model.Conversation, error) {
	var conversation model.Conversation
	if err := s.db.WithContext(ctx).
		Preload("Customer").
		Preload("Channel").
		Preload("AssignedAgent").
		Where("tenant_id = ? AND id = ?", tenantID, conversationID).
		First(&conversation).Error; err != nil {
		return nil, err
	}

	var messages []model.Message
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return nil, err
	}
	conversation.Messages = messages

	ticket, err := s.FindTicketByConversation(ctx, tenantID, conversationID)
	if err == nil {
		conversation.Ticket = ticket
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &conversation, nil
}

// ListConversations returns a paginated conversation list for a tenant.
func (s *Store) ListConversations(ctx context.Context, tenantID string, filter ConversationFilter) ([]model.Conversation, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	var total int64
	query := s.db.WithContext(ctx).Model(&model.Conversation{}).Where("tenant_id = ?", tenantID)
	query = applyConversationFilter(query, filter)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var conversations []model.Conversation
	if err := query.
		Preload("Customer").
		Preload("Channel").
		Preload("AssignedAgent").
		Order("updated_at DESC").
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&conversations).Error; err != nil {
		return nil, 0, err
	}

	return conversations, total, nil
}

// FindTicketByConversation loads the ticket linked to a conversation.
func (s *Store) FindTicketByConversation(ctx context.Context, tenantID string, conversationID string) (*model.Ticket, error) {
	var ticket model.Ticket
	if err := s.db.WithContext(ctx).
		Preload("AssignedAgent").
		Where("tenant_id = ? AND conversation_id = ?", tenantID, conversationID).
		First(&ticket).Error; err != nil {
		return nil, err
	}

	return &ticket, nil
}

// CreateTicket inserts a new ticket.
func (s *Store) CreateTicket(ctx context.Context, ticket *model.Ticket) error {
	return s.db.WithContext(ctx).Create(ticket).Error
}

// SaveTicket updates an existing ticket.
func (s *Store) SaveTicket(ctx context.Context, ticket *model.Ticket) error {
	return s.db.WithContext(ctx).Save(ticket).Error
}

// GetTicketByID loads one ticket with related records.
func (s *Store) GetTicketByID(ctx context.Context, tenantID string, ticketID string) (*model.Ticket, error) {
	var ticket model.Ticket
	if err := s.db.WithContext(ctx).
		Preload("AssignedAgent").
		Where("tenant_id = ? AND id = ?", tenantID, ticketID).
		First(&ticket).Error; err != nil {
		return nil, err
	}

	return &ticket, nil
}

// ListTickets returns a paginated ticket list for a tenant.
func (s *Store) ListTickets(ctx context.Context, tenantID string, filter TicketFilter) ([]model.Ticket, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	var total int64
	query := s.db.WithContext(ctx).Model(&model.Ticket{}).Where("tenant_id = ?", tenantID)
	query = applyTicketFilter(query, filter)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tickets []model.Ticket
	if err := query.
		Preload("AssignedAgent").
		Order("updated_at DESC").
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&tickets).Error; err != nil {
		return nil, 0, err
	}

	return tickets, total, nil
}

// CreateActivityLog inserts a new activity log record.
func (s *Store) CreateActivityLog(ctx context.Context, activityLog *model.ActivityLog) error {
	return s.db.WithContext(ctx).Create(activityLog).Error
}

// CreateOutboxEvent inserts a new outbox event record.
func (s *Store) CreateOutboxEvent(ctx context.Context, outboxEvent *model.OutboxEvent) error {
	return s.db.WithContext(ctx).Create(outboxEvent).Error
}

// ListPendingOutbox returns unpublished outbox events up to the given limit.
func (s *Store) ListPendingOutbox(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	var events []model.OutboxEvent
	if err := s.db.WithContext(ctx).
		Where("published_at IS NULL").
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, err
	}

	return events, nil
}

// MarkOutboxPublished marks an outbox event as published.
func (s *Store) MarkOutboxPublished(ctx context.Context, eventID string) error {
	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&model.OutboxEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]any{
			"published_at": now,
			"failed_at":    nil,
		}).Error
}

// MarkOutboxFailed marks an outbox event as failed and increments its attempt count.
func (s *Store) MarkOutboxFailed(ctx context.Context, eventID string) error {
	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&model.OutboxEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]any{
			"failed_at": now,
			"attempts":  gorm.Expr("attempts + 1"),
		}).Error
}

func applyConversationFilter(query *gorm.DB, filter ConversationFilter) *gorm.DB {
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.AssignedAgentID != "" {
		query = query.Where("assigned_agent_id = ?", filter.AssignedAgentID)
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	return query
}

func applyTicketFilter(query *gorm.DB, filter TicketFilter) *gorm.DB {
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Priority != "" {
		query = query.Where("priority = ?", filter.Priority)
	}
	if filter.AssignedAgentID != "" {
		query = query.Where("assigned_agent_id = ?", filter.AssignedAgentID)
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	return query
}

// DebugTenantSummary returns a compact count summary for one tenant.
func (s *Store) DebugTenantSummary(ctx context.Context, tenantID string) (string, error) {
	var conversations int64
	var tickets int64
	if err := s.db.WithContext(ctx).Model(&model.Conversation{}).Where("tenant_id = ?", tenantID).Count(&conversations).Error; err != nil {
		return "", err
	}
	if err := s.db.WithContext(ctx).Model(&model.Ticket{}).Where("tenant_id = ?", tenantID).Count(&tickets).Error; err != nil {
		return "", err
	}

	return fmt.Sprintf("tenant=%s conversations=%d tickets=%d", tenantID, conversations, tickets), nil
}

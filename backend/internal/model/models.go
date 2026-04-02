package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoleAdmin and the related constants define supported roles, conversation states, sender types, and ticket states.
const (
	RoleAdmin = "admin"
	RoleAgent = "agent"

	ConversationStatusOpen     = "open"
	ConversationStatusAssigned = "assigned"
	ConversationStatusClosed   = "closed"

	SenderTypeCustomer = "customer"
	SenderTypeAgent    = "agent"

	TicketStatusOpen       = "open"
	TicketStatusInProgress = "in_progress"
	TicketStatusResolved   = "resolved"
	TicketStatusClosed     = "closed"

	TicketPriorityLow    = "low"
	TicketPriorityMedium = "medium"
	TicketPriorityHigh   = "high"
)

// BaseModel contains the shared primary key and timestamps for persisted entities.
type BaseModel struct {
	ID        string    `json:"id" gorm:"type:char(36);primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate fills the identifier for models that do not have one yet.
func (m *BaseModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}

	return nil
}

// Tenant identifies one customer workspace in the multi-tenant system.
type Tenant struct {
	BaseModel
	Name string `json:"name" gorm:"size:120;not null"`
	Slug string `json:"slug" gorm:"size:120;not null;uniqueIndex"`
}

// User represents an authenticated admin or agent within a tenant.
type User struct {
	BaseModel
	TenantID     string `json:"tenant_id" gorm:"type:char(36);not null;index"`
	Name         string `json:"name" gorm:"size:120;not null"`
	Email        string `json:"email" gorm:"size:160;not null;uniqueIndex"`
	PasswordHash string `json:"-" gorm:"size:255;not null"`
	Role         string `json:"role" gorm:"size:32;not null;index"`
	Active       bool   `json:"active" gorm:"not null;default:true"`
	Tenant       Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID;references:ID"`
}

// Channel represents one inbound or outbound support channel for a tenant.
type Channel struct {
	BaseModel
	TenantID string `json:"tenant_id" gorm:"type:char(36);not null;index"`
	Key      string `json:"key" gorm:"size:64;not null;index"`
	Name     string `json:"name" gorm:"size:120;not null"`
}

// Customer represents one external customer inside a tenant.
type Customer struct {
	BaseModel
	TenantID   string `json:"tenant_id" gorm:"type:char(36);not null;index"`
	ExternalID string `json:"external_id" gorm:"size:160;not null;index"`
	Name       string `json:"name" gorm:"size:120;not null"`
}

// Conversation tracks an ongoing support thread for a customer and channel.
type Conversation struct {
	BaseModel
	TenantID        string    `json:"tenant_id" gorm:"type:char(36);not null;index"`
	CustomerID      string    `json:"customer_id" gorm:"type:char(36);not null;index"`
	ChannelID       string    `json:"channel_id" gorm:"type:char(36);not null;index"`
	Status          string    `json:"status" gorm:"size:32;not null;index"`
	AssignedAgentID *string   `json:"assigned_agent_id,omitempty" gorm:"type:char(36);index"`
	Customer        Customer  `json:"customer,omitempty" gorm:"foreignKey:CustomerID;references:ID"`
	Channel         Channel   `json:"channel,omitempty" gorm:"foreignKey:ChannelID;references:ID"`
	AssignedAgent   *User     `json:"assigned_agent,omitempty" gorm:"foreignKey:AssignedAgentID;references:ID"`
	Messages        []Message `json:"messages,omitempty" gorm:"foreignKey:ConversationID;references:ID"`
	Ticket          *Ticket   `json:"ticket,omitempty" gorm:"foreignKey:ConversationID;references:ID"`
}

// Message stores one inbound or outbound message in a conversation.
type Message struct {
	BaseModel
	ConversationID string  `json:"conversation_id" gorm:"type:char(36);not null;index"`
	SenderType     string  `json:"sender_type" gorm:"size:32;not null"`
	SenderID       *string `json:"sender_id,omitempty" gorm:"type:char(36);index"`
	Message        string  `json:"message" gorm:"type:text;not null"`
}

// Ticket tracks escalated work linked to a conversation.
type Ticket struct {
	BaseModel
	TenantID        string  `json:"tenant_id" gorm:"type:char(36);not null;index"`
	ConversationID  string  `json:"conversation_id" gorm:"type:char(36);not null;uniqueIndex"`
	Title           string  `json:"title" gorm:"size:180;not null"`
	Description     string  `json:"description" gorm:"type:text;not null"`
	Status          string  `json:"status" gorm:"size:32;not null;index"`
	Priority        string  `json:"priority" gorm:"size:32;not null;index"`
	AssignedAgentID *string `json:"assigned_agent_id,omitempty" gorm:"type:char(36);index"`
	AssignedAgent   *User   `json:"assigned_agent,omitempty" gorm:"foreignKey:AssignedAgentID;references:ID"`
}

// ActivityLog stores a tenant-scoped audit event.
type ActivityLog struct {
	BaseModel
	TenantID   string `json:"tenant_id" gorm:"type:char(36);not null;index"`
	EventType  string `json:"event_type" gorm:"size:120;not null;index"`
	EntityType string `json:"entity_type" gorm:"size:120;not null;index"`
	EntityID   string `json:"entity_id" gorm:"type:char(36);not null;index"`
	Payload    string `json:"payload" gorm:"type:longtext;not null"`
}

// OutboxEvent stores a domain event waiting to be published asynchronously.
type OutboxEvent struct {
	BaseModel
	TenantID    string     `json:"tenant_id" gorm:"type:char(36);not null;index"`
	EventType   string     `json:"event_type" gorm:"size:120;not null;index"`
	EntityType  string     `json:"entity_type" gorm:"size:120;not null;index"`
	EntityID    string     `json:"entity_id" gorm:"type:char(36);not null;index"`
	RoutingKey  string     `json:"routing_key" gorm:"size:160;not null;index"`
	Payload     string     `json:"payload" gorm:"type:longtext;not null"`
	Attempts    int        `json:"attempts" gorm:"not null;default:0"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
}

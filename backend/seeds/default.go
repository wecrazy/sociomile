package seeds

import (
	"context"

	"github.com/wecrazy/sociomile/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LoadDemoData inserts or updates the demo tenants, users, channels, and records.
func LoadDemoData(ctx context.Context, db *gorm.DB) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	tenants := []model.Tenant{
		{BaseModel: model.BaseModel{ID: "11111111-1111-1111-1111-111111111111"}, Name: "Acme Support", Slug: "acme-support"},
		{BaseModel: model.BaseModel{ID: "22222222-2222-2222-2222-222222222222"}, Name: "Globex Care", Slug: "globex-care"},
	}
	if err := upsert(ctx, db, &tenants, []string{"name", "slug", "updated_at"}); err != nil {
		return err
	}

	users := []model.User{
		{BaseModel: model.BaseModel{ID: "11111111-aaaa-aaaa-aaaa-111111111111"}, TenantID: tenants[0].ID, Name: "Alice Admin", Email: "alice.admin@acme.local", PasswordHash: string(passwordHash), Role: model.RoleAdmin, Active: true},
		{BaseModel: model.BaseModel{ID: "11111111-bbbb-bbbb-bbbb-111111111111"}, TenantID: tenants[0].ID, Name: "Aaron Agent", Email: "aaron.agent@acme.local", PasswordHash: string(passwordHash), Role: model.RoleAgent, Active: true},
		{BaseModel: model.BaseModel{ID: "22222222-aaaa-aaaa-aaaa-222222222222"}, TenantID: tenants[1].ID, Name: "Grace Admin", Email: "grace.admin@globex.local", PasswordHash: string(passwordHash), Role: model.RoleAdmin, Active: true},
		{BaseModel: model.BaseModel{ID: "22222222-bbbb-bbbb-bbbb-222222222222"}, TenantID: tenants[1].ID, Name: "Gina Agent", Email: "gina.agent@globex.local", PasswordHash: string(passwordHash), Role: model.RoleAgent, Active: true},
	}
	if err := upsert(ctx, db, &users, []string{"tenant_id", "name", "password_hash", "role", "active", "updated_at"}); err != nil {
		return err
	}

	channels := []model.Channel{
		{BaseModel: model.BaseModel{ID: "31111111-1111-1111-1111-111111111111"}, TenantID: tenants[0].ID, Key: "whatsapp", Name: "WhatsApp"},
		{BaseModel: model.BaseModel{ID: "32222222-2222-2222-2222-222222222222"}, TenantID: tenants[0].ID, Key: "instagram", Name: "Instagram"},
		{BaseModel: model.BaseModel{ID: "41111111-1111-1111-1111-111111111111"}, TenantID: tenants[1].ID, Key: "whatsapp", Name: "WhatsApp"},
		{BaseModel: model.BaseModel{ID: "42222222-2222-2222-2222-222222222222"}, TenantID: tenants[1].ID, Key: "instagram", Name: "Instagram"},
	}
	if err := upsert(ctx, db, &channels, []string{"tenant_id", "key", "name", "updated_at"}); err != nil {
		return err
	}

	customers := []model.Customer{
		{BaseModel: model.BaseModel{ID: "51111111-1111-1111-1111-111111111111"}, TenantID: tenants[0].ID, ExternalID: "wa-acme-20481", Name: "Lena Hart"},
		{BaseModel: model.BaseModel{ID: "52222222-2222-2222-2222-222222222222"}, TenantID: tenants[1].ID, ExternalID: "ig-globex-88014", Name: "Rafi Noor"},
	}
	if err := upsert(ctx, db, &customers, []string{"tenant_id", "external_id", "name", "updated_at"}); err != nil {
		return err
	}

	assignedAgentID := users[3].ID
	conversations := []model.Conversation{
		{BaseModel: model.BaseModel{ID: "61111111-1111-1111-1111-111111111111"}, TenantID: tenants[0].ID, CustomerID: customers[0].ID, ChannelID: channels[0].ID, Status: model.ConversationStatusOpen},
		{BaseModel: model.BaseModel{ID: "62222222-2222-2222-2222-222222222222"}, TenantID: tenants[1].ID, CustomerID: customers[1].ID, ChannelID: channels[3].ID, Status: model.ConversationStatusAssigned, AssignedAgentID: &assignedAgentID},
	}
	if err := upsert(ctx, db, &conversations, []string{"tenant_id", "customer_id", "channel_id", "status", "assigned_agent_id", "updated_at"}); err != nil {
		return err
	}

	messageSenderID := users[3].ID
	messages := []model.Message{
		{BaseModel: model.BaseModel{ID: "71111111-1111-1111-1111-111111111111"}, ConversationID: conversations[0].ID, SenderType: model.SenderTypeCustomer, Message: "Hi, order ACM-20481 is marked delivered, but nothing has reached my building yet."},
		{BaseModel: model.BaseModel{ID: "72222222-2222-2222-2222-222222222222"}, ConversationID: conversations[1].ID, SenderType: model.SenderTypeAgent, SenderID: &messageSenderID, Message: "We traced the billing mismatch and escalated it to our subscription operations team."},
	}
	if err := upsert(ctx, db, &messages, []string{"conversation_id", "sender_type", "sender_id", "message", "updated_at"}); err != nil {
		return err
	}

	tickets := []model.Ticket{
		{BaseModel: model.BaseModel{ID: "81111111-1111-1111-1111-111111111111"}, TenantID: tenants[1].ID, ConversationID: conversations[1].ID, Title: "Resolve annual plan billing mismatch", Description: "Customer reported that the renewal invoice total does not match the approved subscription quote.", Status: model.TicketStatusInProgress, Priority: model.TicketPriorityHigh, AssignedAgentID: &assignedAgentID},
	}
	return upsert(ctx, db, &tickets, []string{"tenant_id", "conversation_id", "title", "description", "status", "priority", "assigned_agent_id", "updated_at"})
}

func upsert[T any](ctx context.Context, db *gorm.DB, values *[]T, columns []string) error {
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns(columns),
	}).Create(values).Error
}

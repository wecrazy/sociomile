package repository

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOutboxHelpersAndTenantSummary(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "store-test.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Tenant{},
		&model.User{},
		&model.Channel{},
		&model.Customer{},
		&model.Conversation{},
		&model.Message{},
		&model.Ticket{},
		&model.ActivityLog{},
		&model.OutboxEvent{},
	))

	ctx := context.Background()
	store := NewStore(db)
	require.NotNil(t, store.DB())

	tenant := model.Tenant{BaseModel: model.BaseModel{ID: "tenant-1"}, Name: "Tenant One", Slug: "tenant-one"}
	channel := model.Channel{BaseModel: model.BaseModel{ID: "channel-1"}, TenantID: tenant.ID, Key: "whatsapp", Name: "WhatsApp"}
	customer := model.Customer{BaseModel: model.BaseModel{ID: "customer-1"}, TenantID: tenant.ID, ExternalID: "customer-ext-1", Name: "Customer One"}
	conversation := model.Conversation{BaseModel: model.BaseModel{ID: "conversation-1"}, TenantID: tenant.ID, CustomerID: customer.ID, ChannelID: channel.ID, Status: model.ConversationStatusOpen}
	ticket := model.Ticket{BaseModel: model.BaseModel{ID: "ticket-1"}, TenantID: tenant.ID, ConversationID: conversation.ID, Title: "Ticket One", Description: "Needs work", Status: model.TicketStatusOpen, Priority: model.TicketPriorityMedium}
	outboxEvent := model.OutboxEvent{BaseModel: model.BaseModel{ID: "event-1"}, TenantID: tenant.ID, EventType: "ticket.created", EntityType: "ticket", EntityID: ticket.ID, RoutingKey: "ticket.created", Payload: "{}"}

	require.NoError(t, db.Create(&tenant).Error)
	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, db.Create(&customer).Error)
	require.NoError(t, db.Create(&conversation).Error)
	require.NoError(t, db.Create(&ticket).Error)
	require.NoError(t, db.Create(&outboxEvent).Error)

	pending, err := store.ListPendingOutbox(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, outboxEvent.ID, pending[0].ID)

	require.NoError(t, store.MarkOutboxFailed(ctx, outboxEvent.ID))

	var failed model.OutboxEvent
	require.NoError(t, db.First(&failed, "id = ?", outboxEvent.ID).Error)
	require.Equal(t, 1, failed.Attempts)
	require.NotNil(t, failed.FailedAt)

	require.NoError(t, store.MarkOutboxPublished(ctx, outboxEvent.ID))

	var published model.OutboxEvent
	require.NoError(t, db.First(&published, "id = ?", outboxEvent.ID).Error)
	require.NotNil(t, published.PublishedAt)
	require.Nil(t, published.FailedAt)

	summary, err := store.DebugTenantSummary(ctx, tenant.ID)
	require.NoError(t, err)
	require.Contains(t, summary, "tenant=tenant-1")
	require.Contains(t, summary, "conversations=1")
	require.Contains(t, summary, "tickets=1")
}

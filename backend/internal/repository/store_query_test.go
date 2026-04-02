package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type repositoryFixture struct {
	ctx                  context.Context
	db                   *gorm.DB
	store                *Store
	tenant               model.Tenant
	otherTenant          model.Tenant
	admin                model.User
	agent                model.User
	inactiveAgent        model.User
	channel              model.Channel
	customer             model.Customer
	openConversation     model.Conversation
	assignedConversation model.Conversation
	closedConversation   model.Conversation
	ticket               model.Ticket
	outboxEvent          model.OutboxEvent
}

func TestStoreQueriesAndFilters(t *testing.T) {
	fixture := newRepositoryFixture(t)

	tenant, err := fixture.store.GetTenant(fixture.ctx, fixture.tenant.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.tenant.ID, tenant.ID)
	_, err = fixture.store.GetTenant(fixture.ctx, "missing")
	require.Error(t, err)

	user, err := fixture.store.GetUserByEmail(fixture.ctx, fixture.admin.Email)
	require.NoError(t, err)
	require.Equal(t, fixture.admin.ID, user.ID)
	require.Equal(t, fixture.tenant.ID, user.Tenant.ID)
	_, err = fixture.store.GetUserByEmail(fixture.ctx, "missing@example.com")
	require.Error(t, err)

	user, err = fixture.store.GetUserByIDAndTenant(fixture.ctx, fixture.agent.ID, fixture.tenant.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.agent.ID, user.ID)

	agents, err := fixture.store.ListUsersByRole(fixture.ctx, fixture.tenant.ID, model.RoleAgent)
	require.NoError(t, err)
	require.Len(t, agents, 1)
	require.Equal(t, fixture.agent.ID, agents[0].ID)

	channel, err := fixture.store.GetChannelByKey(fixture.ctx, fixture.tenant.ID, fixture.channel.Key)
	require.NoError(t, err)
	require.Equal(t, fixture.channel.ID, channel.ID)
	_, err = fixture.store.GetChannelByKey(fixture.ctx, fixture.tenant.ID, "missing")
	require.Error(t, err)

	updatedCustomer, err := fixture.store.GetOrCreateCustomer(fixture.ctx, fixture.tenant.ID, fixture.customer.ExternalID, "Renamed Customer")
	require.NoError(t, err)
	require.Equal(t, "Renamed Customer", updatedCustomer.Name)

	unchangedCustomer, err := fixture.store.GetOrCreateCustomer(fixture.ctx, fixture.tenant.ID, fixture.customer.ExternalID, "Renamed Customer")
	require.NoError(t, err)
	require.Equal(t, "Renamed Customer", unchangedCustomer.Name)

	createdCustomer, err := fixture.store.GetOrCreateCustomer(fixture.ctx, fixture.tenant.ID, "external-2", "")
	require.NoError(t, err)
	require.Equal(t, "external-2", createdCustomer.Name)

	activeConversation, err := fixture.store.FindActiveConversation(fixture.ctx, fixture.tenant.ID, fixture.customer.ID, fixture.channel.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.assignedConversation.ID, activeConversation.ID)
	_, err = fixture.store.FindActiveConversation(fixture.ctx, fixture.tenant.ID, "missing", fixture.channel.ID)
	require.Error(t, err)

	conversation, err := fixture.store.GetConversationByID(fixture.ctx, fixture.tenant.ID, fixture.assignedConversation.ID)
	require.NoError(t, err)
	require.Len(t, conversation.Messages, 2)
	require.Equal(t, "First reply", conversation.Messages[0].Message)
	require.Equal(t, "Second reply", conversation.Messages[1].Message)
	require.NotNil(t, conversation.Ticket)
	conversation, err = fixture.store.GetConversationByID(fixture.ctx, fixture.tenant.ID, fixture.openConversation.ID)
	require.NoError(t, err)
	require.Nil(t, conversation.Ticket)

	conversations, total, err := fixture.store.ListConversations(fixture.ctx, fixture.tenant.ID, ConversationFilter{
		Status:          model.ConversationStatusAssigned,
		AssignedAgentID: fixture.agent.ID,
		Offset:          0,
		Limit:           0,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, conversations, 1)
	require.Equal(t, fixture.assignedConversation.ID, conversations[0].ID)

	conversations, total, err = fixture.store.ListConversations(fixture.ctx, fixture.tenant.ID, ConversationFilter{
		Offset: 0,
		Limit:  1,
	})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, conversations, 1)
	require.NotEmpty(t, conversations[0].ID)

	conversations, total, err = fixture.store.ListConversations(fixture.ctx, fixture.tenant.ID, ConversationFilter{
		AssignedAgentID: fixture.agent.ID,
		Offset:          0,
		Limit:           5,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, conversations, 1)
	require.Equal(t, fixture.assignedConversation.ID, conversations[0].ID)

	ticket, err := fixture.store.GetTicketByID(fixture.ctx, fixture.tenant.ID, fixture.ticket.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.ticket.ID, ticket.ID)
	_, err = fixture.store.GetTicketByID(fixture.ctx, fixture.tenant.ID, "missing")
	require.Error(t, err)

	tickets, total, err := fixture.store.ListTickets(fixture.ctx, fixture.tenant.ID, TicketFilter{
		Status:   model.TicketStatusOpen,
		Priority: model.TicketPriorityHigh,
		Offset:   0,
		Limit:    0,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, tickets, 1)
	require.Equal(t, fixture.ticket.ID, tickets[0].ID)

	tickets, total, err = fixture.store.ListTickets(fixture.ctx, fixture.tenant.ID, TicketFilter{
		AssignedAgentID: fixture.agent.ID,
		Offset:          0,
		Limit:           1,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, tickets, 1)
	require.Equal(t, fixture.ticket.ID, tickets[0].ID)

	var filteredConversations []model.Conversation
	conversationStatement := applyConversationFilter(
		fixture.db.Session(&gorm.Session{DryRun: true}).Model(&model.Conversation{}),
		ConversationFilter{Status: model.ConversationStatusAssigned, AssignedAgentID: fixture.agent.ID},
	).Find(&filteredConversations).Statement
	require.Contains(t, conversationStatement.SQL.String(), "status = ?")
	require.Contains(t, conversationStatement.SQL.String(), "assigned_agent_id")

	var filteredTickets []model.Ticket
	statement := applyTicketFilter(
		fixture.db.Session(&gorm.Session{DryRun: true}).Model(&model.Ticket{}),
		TicketFilter{AssignedAgentID: fixture.agent.ID, Limit: 0},
	).Find(&filteredTickets).Statement
	require.Contains(t, statement.SQL.String(), "assigned_agent_id")

	pending, err := fixture.store.ListPendingOutbox(fixture.ctx, 1)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, fixture.outboxEvent.ID, pending[0].ID)

	summary, err := fixture.store.DebugTenantSummary(fixture.ctx, fixture.tenant.ID)
	require.NoError(t, err)
	require.Contains(t, summary, "tenant=tenant-1")
	require.Contains(t, summary, "conversations=3")
	require.Contains(t, summary, "tickets=1")
}

func TestStoreReturnsDatabaseErrorsAfterClose(t *testing.T) {
	fixture := newRepositoryFixture(t)

	sqlDB, err := fixture.db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = fixture.store.ListPendingOutbox(fixture.ctx, 10)
	require.Error(t, err)
	_, err = fixture.store.GetOrCreateCustomer(fixture.ctx, fixture.tenant.ID, fixture.customer.ExternalID, "Renamed Again")
	require.Error(t, err)
	_, err = fixture.store.GetConversationByID(fixture.ctx, fixture.tenant.ID, fixture.assignedConversation.ID)
	require.Error(t, err)
	_, _, err = fixture.store.ListConversations(fixture.ctx, fixture.tenant.ID, ConversationFilter{})
	require.Error(t, err)
	_, err = fixture.store.GetTicketByID(fixture.ctx, fixture.tenant.ID, fixture.ticket.ID)
	require.Error(t, err)
	_, _, err = fixture.store.ListTickets(fixture.ctx, fixture.tenant.ID, TicketFilter{})
	require.Error(t, err)
	_, err = fixture.store.DebugTenantSummary(fixture.ctx, fixture.tenant.ID)
	require.Error(t, err)
}

func TestStoreMutationHelpersReturnDatabaseErrorsAfterClose(t *testing.T) {
	fixture := newRepositoryFixture(t)

	sqlDB, err := fixture.db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	conversation := &model.Conversation{BaseModel: model.BaseModel{ID: "closed-conversation"}, TenantID: fixture.tenant.ID, CustomerID: fixture.customer.ID, ChannelID: fixture.channel.ID, Status: model.ConversationStatusOpen}
	require.Error(t, fixture.store.CreateConversation(fixture.ctx, conversation))
	require.Error(t, fixture.store.SaveConversation(fixture.ctx, &fixture.openConversation))
	require.Error(t, fixture.store.CreateMessage(fixture.ctx, &model.Message{BaseModel: model.BaseModel{ID: "closed-message"}, ConversationID: fixture.openConversation.ID, SenderType: model.SenderTypeCustomer, Message: "hello"}))
	require.Error(t, fixture.store.CreateTicket(fixture.ctx, &model.Ticket{BaseModel: model.BaseModel{ID: "closed-ticket"}, TenantID: fixture.tenant.ID, ConversationID: fixture.openConversation.ID, Title: "ticket", Description: "desc", Status: model.TicketStatusOpen, Priority: model.TicketPriorityLow}))
	require.Error(t, fixture.store.SaveTicket(fixture.ctx, &fixture.ticket))
	require.Error(t, fixture.store.CreateActivityLog(fixture.ctx, &model.ActivityLog{BaseModel: model.BaseModel{ID: "closed-activity"}, TenantID: fixture.tenant.ID, EventType: "event", EntityType: "conversation", EntityID: fixture.openConversation.ID, Payload: "{}"}))
	require.Error(t, fixture.store.CreateOutboxEvent(fixture.ctx, &model.OutboxEvent{BaseModel: model.BaseModel{ID: "closed-event"}, TenantID: fixture.tenant.ID, EventType: "event", EntityType: "conversation", EntityID: fixture.openConversation.ID, RoutingKey: "event", Payload: "{}"}))
	require.Error(t, fixture.store.MarkOutboxPublished(fixture.ctx, fixture.outboxEvent.ID))
	require.Error(t, fixture.store.MarkOutboxFailed(fixture.ctx, fixture.outboxEvent.ID))
}

func TestStoreCreateAndMutationHelpers(t *testing.T) {
	fixture := newRepositoryFixture(t)
	agentID := fixture.agent.ID

	newCustomer := model.Customer{
		BaseModel:  model.BaseModel{ID: "customer-2"},
		TenantID:   fixture.tenant.ID,
		ExternalID: "external-3",
		Name:       "Customer Three",
	}
	require.NoError(t, fixture.db.Create(&newCustomer).Error)

	conversation := &model.Conversation{
		BaseModel:  model.BaseModel{ID: "conversation-created"},
		TenantID:   fixture.tenant.ID,
		CustomerID: newCustomer.ID,
		ChannelID:  fixture.channel.ID,
		Status:     model.ConversationStatusOpen,
	}
	require.NoError(t, fixture.store.CreateConversation(fixture.ctx, conversation))

	conversation.Status = model.ConversationStatusAssigned
	conversation.AssignedAgentID = &agentID
	require.NoError(t, fixture.store.SaveConversation(fixture.ctx, conversation))

	message := &model.Message{
		BaseModel:      model.BaseModel{ID: "message-created"},
		ConversationID: conversation.ID,
		SenderType:     model.SenderTypeCustomer,
		Message:        "Created through store",
	}
	require.NoError(t, fixture.store.CreateMessage(fixture.ctx, message))

	_, err := fixture.store.FindTicketByConversation(fixture.ctx, fixture.tenant.ID, conversation.ID)
	require.Error(t, err)

	ticket := &model.Ticket{
		BaseModel:       model.BaseModel{ID: "ticket-created"},
		TenantID:        fixture.tenant.ID,
		ConversationID:  conversation.ID,
		Title:           "Created Ticket",
		Description:     "Created through store",
		Status:          model.TicketStatusOpen,
		Priority:        model.TicketPriorityLow,
		AssignedAgentID: &agentID,
	}
	require.NoError(t, fixture.store.CreateTicket(fixture.ctx, ticket))

	ticket.Status = model.TicketStatusResolved
	require.NoError(t, fixture.store.SaveTicket(fixture.ctx, ticket))

	foundTicket, err := fixture.store.FindTicketByConversation(fixture.ctx, fixture.tenant.ID, conversation.ID)
	require.NoError(t, err)
	require.Equal(t, ticket.ID, foundTicket.ID)

	activityLog := &model.ActivityLog{
		BaseModel:  model.BaseModel{ID: "activity-created"},
		TenantID:   fixture.tenant.ID,
		EventType:  "conversation.updated",
		EntityType: "conversation",
		EntityID:   conversation.ID,
		Payload:    "{}",
	}
	require.NoError(t, fixture.store.CreateActivityLog(fixture.ctx, activityLog))

	outboxEvent := &model.OutboxEvent{
		BaseModel:  model.BaseModel{ID: "event-created"},
		TenantID:   fixture.tenant.ID,
		EventType:  "conversation.updated",
		EntityType: "conversation",
		EntityID:   conversation.ID,
		RoutingKey: "conversation.updated",
		Payload:    "{}",
	}
	require.NoError(t, fixture.store.CreateOutboxEvent(fixture.ctx, outboxEvent))
	require.NoError(t, fixture.store.MarkOutboxFailed(fixture.ctx, outboxEvent.ID))
	require.NoError(t, fixture.store.MarkOutboxPublished(fixture.ctx, outboxEvent.ID))

	var persistedEvent model.OutboxEvent
	require.NoError(t, fixture.db.First(&persistedEvent, "id = ?", outboxEvent.ID).Error)
	require.Equal(t, 1, persistedEvent.Attempts)
	require.NotNil(t, persistedEvent.PublishedAt)
	require.Nil(t, persistedEvent.FailedAt)
}

func newRepositoryFixture(t *testing.T) *repositoryFixture {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "repository.db")), &gorm.Config{})
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

	now := time.Now().UTC()
	assignedAgentID := "agent-1"
	fixture := &repositoryFixture{
		ctx:                  context.Background(),
		db:                   db,
		store:                NewStore(db),
		tenant:               model.Tenant{BaseModel: model.BaseModel{ID: "tenant-1", CreatedAt: now, UpdatedAt: now}, Name: "Tenant One", Slug: "tenant-one"},
		otherTenant:          model.Tenant{BaseModel: model.BaseModel{ID: "tenant-2", CreatedAt: now, UpdatedAt: now}, Name: "Tenant Two", Slug: "tenant-two"},
		admin:                model.User{BaseModel: model.BaseModel{ID: "admin-1", CreatedAt: now, UpdatedAt: now}, TenantID: "tenant-1", Name: "Admin", Email: "admin@example.com", PasswordHash: "hash", Role: model.RoleAdmin, Active: true},
		agent:                model.User{BaseModel: model.BaseModel{ID: "agent-1", CreatedAt: now, UpdatedAt: now.Add(time.Minute)}, TenantID: "tenant-1", Name: "Agent", Email: "agent@example.com", PasswordHash: "hash", Role: model.RoleAgent, Active: true},
		inactiveAgent:        model.User{BaseModel: model.BaseModel{ID: "agent-2", CreatedAt: now, UpdatedAt: now}, TenantID: "tenant-1", Name: "Inactive Agent", Email: "inactive@example.com", PasswordHash: "hash", Role: model.RoleAgent, Active: false},
		channel:              model.Channel{BaseModel: model.BaseModel{ID: "channel-1", CreatedAt: now, UpdatedAt: now}, TenantID: "tenant-1", Key: "whatsapp", Name: "WhatsApp"},
		customer:             model.Customer{BaseModel: model.BaseModel{ID: "customer-1", CreatedAt: now, UpdatedAt: now}, TenantID: "tenant-1", ExternalID: "external-1", Name: "Customer One"},
		openConversation:     model.Conversation{BaseModel: model.BaseModel{ID: "conversation-open", CreatedAt: now, UpdatedAt: now}, TenantID: "tenant-1", CustomerID: "customer-1", ChannelID: "channel-1", Status: model.ConversationStatusOpen},
		assignedConversation: model.Conversation{BaseModel: model.BaseModel{ID: "conversation-assigned", CreatedAt: now, UpdatedAt: now.Add(2 * time.Minute)}, TenantID: "tenant-1", CustomerID: "customer-1", ChannelID: "channel-1", Status: model.ConversationStatusAssigned, AssignedAgentID: &assignedAgentID},
		closedConversation:   model.Conversation{BaseModel: model.BaseModel{ID: "conversation-closed", CreatedAt: now, UpdatedAt: now.Add(3 * time.Minute)}, TenantID: "tenant-1", CustomerID: "customer-1", ChannelID: "channel-1", Status: model.ConversationStatusClosed},
		ticket:               model.Ticket{BaseModel: model.BaseModel{ID: "ticket-1", CreatedAt: now, UpdatedAt: now}, TenantID: "tenant-1", ConversationID: "conversation-assigned", Title: "Ticket One", Description: "Needs work", Status: model.TicketStatusOpen, Priority: model.TicketPriorityHigh, AssignedAgentID: &assignedAgentID},
		outboxEvent:          model.OutboxEvent{BaseModel: model.BaseModel{ID: "event-1", CreatedAt: now, UpdatedAt: now}, TenantID: "tenant-1", EventType: "ticket.created", EntityType: "ticket", EntityID: "ticket-1", RoutingKey: "ticket.created", Payload: "{}"},
	}

	for _, value := range []any{
		&fixture.tenant,
		&fixture.otherTenant,
		&fixture.admin,
		&fixture.agent,
		&fixture.inactiveAgent,
		&fixture.channel,
		&fixture.customer,
		&fixture.openConversation,
		&fixture.assignedConversation,
		&fixture.closedConversation,
		&model.Message{BaseModel: model.BaseModel{ID: "message-1", CreatedAt: now, UpdatedAt: now}, ConversationID: fixture.assignedConversation.ID, SenderType: model.SenderTypeAgent, SenderID: &assignedAgentID, Message: "First reply"},
		&model.Message{BaseModel: model.BaseModel{ID: "message-2", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)}, ConversationID: fixture.assignedConversation.ID, SenderType: model.SenderTypeCustomer, Message: "Second reply"},
		&fixture.ticket,
		&fixture.outboxEvent,
	} {
		require.NoError(t, db.Create(value).Error)
	}
	require.NoError(t, db.Model(&fixture.inactiveAgent).Update("active", false).Error)
	require.NoError(t, db.Model(&fixture.assignedConversation).Update("assigned_agent_id", assignedAgentID).Error)
	require.NoError(t, db.Model(&fixture.ticket).Update("assigned_agent_id", assignedAgentID).Error)

	return fixture
}

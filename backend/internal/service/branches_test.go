package service

import (
	"context"
	"path/filepath"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/apperror"
	"github.com/wecrazy/sociomile/backend/internal/cache"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"github.com/wecrazy/sociomile/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuthServiceAdditionalBranches(t *testing.T) {
	fixture := newServiceFixture(t)

	fixture.adminA.Active = false
	require.NoError(t, fixture.store.DB().Save(&fixture.adminA).Error)
	_, err := fixture.authService.Login(fixture.ctx, fixture.adminA.Email, fixture.password)
	requireAppErrorCode(t, err, "inactive_user")

	_, err = fixture.authService.GetMe(fixture.ctx, fixture.adminA.ID, fixture.tenantB.ID)
	require.Error(t, err)
}

func TestConversationServiceValidationBranches(t *testing.T) {
	fixture := newServiceFixture(t)

	err := fixture.conversationService.AllowWebhook(fixture.ctx, "", "127.0.0.1")
	requireAppErrorCode(t, err, "invalid_tenant")

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	rateLimitedService := NewConversationService(fixture.store, cache.New(client))
	for index := 0; index < 30; index++ {
		require.NoError(t, rateLimitedService.AllowWebhook(fixture.ctx, fixture.tenantA.ID, "127.0.0.1"))
	}
	err = rateLimitedService.AllowWebhook(fixture.ctx, fixture.tenantA.ID, "127.0.0.1")
	requireAppErrorCode(t, err, "rate_limited")
	require.NoError(t, client.Close())
	require.NoError(t, rateLimitedService.AllowWebhook(fixture.ctx, fixture.tenantA.ID, "127.0.0.1"))

	_, err = fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{})
	requireAppErrorCode(t, err, "invalid_webhook")

	_, err = fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           "missing-tenant",
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "customer-1",
		Message:            "hello",
	})
	requireAppErrorCode(t, err, "tenant_not_found")

	_, err = fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         "missing-channel",
		CustomerExternalID: "customer-1",
		Message:            "hello",
	})
	requireAppErrorCode(t, err, "invalid_channel")

	firstConversation, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "repeat-customer",
		CustomerName:       "Original Name",
		Message:            "hello",
	})
	require.NoError(t, err)

	secondConversation, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "repeat-customer",
		CustomerName:       "Renamed Customer",
		Message:            "follow-up",
	})
	require.NoError(t, err)
	require.Equal(t, firstConversation.ID, secondConversation.ID)
	require.Equal(t, "Renamed Customer", secondConversation.Customer.Name)
	require.Len(t, secondConversation.Messages, 2)
}

func TestConversationAssignmentReplyAndCloseBranches(t *testing.T) {
	fixture := newServiceFixture(t)
	extraAgent := createTenantAgent(t, fixture.store, fixture.tenantA.ID, "tenant-a-agent-2")

	conversation, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "assign-1",
		CustomerName:       "Assign Test",
		Message:            "hello",
	})
	require.NoError(t, err)

	_, err = fixture.conversationService.AssignConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.adminB.ID, fixture.adminB.Role, AssignConversationInput{AgentID: fixture.agentA.ID})
	requireAppErrorCode(t, err, "forbidden")
	_, err = fixture.conversationService.AssignConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.adminA.ID, fixture.adminA.Role, AssignConversationInput{})
	requireAppErrorCode(t, err, "invalid_assignment")
	_, err = fixture.conversationService.AssignConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.adminA.ID, fixture.adminA.Role, AssignConversationInput{AgentID: "missing-agent"})
	requireAppErrorCode(t, err, "invalid_assignment")
	_, err = fixture.conversationService.AssignConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.adminA.ID, fixture.adminA.Role, AssignConversationInput{AgentID: fixture.adminA.ID})
	requireAppErrorCode(t, err, "invalid_assignment")

	conversation.Status = model.ConversationStatusClosed
	require.NoError(t, fixture.store.SaveConversation(fixture.ctx, conversation))
	_, err = fixture.conversationService.AssignConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.adminA.ID, fixture.adminA.Role, AssignConversationInput{AgentID: fixture.agentA.ID})
	requireAppErrorCode(t, err, "conversation_closed")

	replyConversation, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "reply-1",
		CustomerName:       "Reply Test",
		Message:            "hello",
	})
	require.NoError(t, err)

	_, err = fixture.conversationService.ReplyToConversation(fixture.ctx, fixture.tenantA.ID, replyConversation.ID, fixture.agentA.ID, ReplyConversationInput{})
	requireAppErrorCode(t, err, "invalid_message")
	_, err = fixture.conversationService.ReplyToConversation(fixture.ctx, fixture.tenantA.ID, replyConversation.ID, fixture.adminA.ID, ReplyConversationInput{Message: "admin reply"})
	requireAppErrorCode(t, err, "forbidden")

	_, err = fixture.conversationService.AssignConversation(fixture.ctx, fixture.tenantA.ID, replyConversation.ID, fixture.adminA.ID, fixture.adminA.Role, AssignConversationInput{AgentID: fixture.agentA.ID})
	require.NoError(t, err)
	_, err = fixture.conversationService.ReplyToConversation(fixture.ctx, fixture.tenantA.ID, replyConversation.ID, extraAgent.ID, ReplyConversationInput{Message: "wrong assignee"})
	requireAppErrorCode(t, err, "conversation_assigned_elsewhere")

	autoAssignedConversation, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "reply-2",
		CustomerName:       "Auto Assign",
		Message:            "hello",
	})
	require.NoError(t, err)

	autoAssignedConversation, err = fixture.conversationService.ReplyToConversation(fixture.ctx, fixture.tenantA.ID, autoAssignedConversation.ID, fixture.agentA.ID, ReplyConversationInput{Message: "handled"})
	require.NoError(t, err)
	require.NotNil(t, autoAssignedConversation.AssignedAgentID)
	require.Equal(t, fixture.agentA.ID, *autoAssignedConversation.AssignedAgentID)

	_, err = fixture.conversationService.CloseConversation(fixture.ctx, fixture.tenantA.ID, autoAssignedConversation.ID, extraAgent.ID, model.RoleAgent)
	requireAppErrorCode(t, err, "forbidden")

	closedConversation, err := fixture.conversationService.CloseConversation(fixture.ctx, fixture.tenantA.ID, autoAssignedConversation.ID, fixture.agentA.ID, model.RoleAgent)
	require.NoError(t, err)
	require.Equal(t, model.ConversationStatusClosed, closedConversation.Status)

	closedConversation, err = fixture.conversationService.CloseConversation(fixture.ctx, fixture.tenantA.ID, autoAssignedConversation.ID, fixture.agentA.ID, model.RoleAgent)
	require.NoError(t, err)
	require.Equal(t, model.ConversationStatusClosed, closedConversation.Status)

	_, err = fixture.conversationService.ReplyToConversation(fixture.ctx, fixture.tenantA.ID, autoAssignedConversation.ID, fixture.agentA.ID, ReplyConversationInput{Message: "after close"})
	requireAppErrorCode(t, err, "conversation_closed")
}

func TestTicketServiceValidationBranches(t *testing.T) {
	fixture := newServiceFixture(t)
	extraAgent := createTenantAgent(t, fixture.store, fixture.tenantA.ID, "tenant-a-agent-3")

	conversation, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "ticket-1",
		CustomerName:       "Ticket Test",
		Message:            "hello",
	})
	require.NoError(t, err)

	_, err = fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.agentA.ID, EscalateTicketInput{})
	requireAppErrorCode(t, err, "invalid_ticket")
	_, err = fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.agentA.ID, EscalateTicketInput{Title: "Need help", Description: "Investigate", Priority: "urgent"})
	requireAppErrorCode(t, err, "invalid_priority")
	_, err = fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, "missing-agent", EscalateTicketInput{Title: "Need help", Description: "Investigate"})
	requireAppErrorCode(t, err, "forbidden")
	_, err = fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.adminA.ID, EscalateTicketInput{Title: "Need help", Description: "Investigate"})
	requireAppErrorCode(t, err, "forbidden")

	ticket, err := fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.agentA.ID, EscalateTicketInput{Title: "Need help", Description: "Investigate"})
	require.NoError(t, err)
	require.Equal(t, model.TicketPriorityMedium, ticket.Priority)

	_, err = fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.agentA.ID, EscalateTicketInput{Title: "Duplicate", Description: "Investigate"})
	requireAppErrorCode(t, err, "ticket_exists")

	assignedElsewhereConversation, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "ticket-2",
		CustomerName:       "Assigned Elsewhere",
		Message:            "hello",
	})
	require.NoError(t, err)
	_, err = fixture.conversationService.AssignConversation(fixture.ctx, fixture.tenantA.ID, assignedElsewhereConversation.ID, fixture.adminA.ID, fixture.adminA.Role, AssignConversationInput{AgentID: extraAgent.ID})
	require.NoError(t, err)
	_, err = fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, assignedElsewhereConversation.ID, fixture.agentA.ID, EscalateTicketInput{Title: "Need help", Description: "Investigate"})
	requireAppErrorCode(t, err, "conversation_assigned_elsewhere")

	closedConversation, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "ticket-3",
		CustomerName:       "Closed Conversation",
		Message:            "hello",
	})
	require.NoError(t, err)
	closedConversation.Status = model.ConversationStatusClosed
	require.NoError(t, fixture.store.SaveConversation(fixture.ctx, closedConversation))
	_, err = fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, closedConversation.ID, fixture.agentA.ID, EscalateTicketInput{Title: "Need help", Description: "Investigate"})
	requireAppErrorCode(t, err, "conversation_closed")

	_, err = fixture.ticketService.UpdateStatus(fixture.ctx, fixture.tenantA.ID, ticket.ID, fixture.adminB.ID, fixture.adminB.Role, UpdateTicketStatusInput{Status: model.TicketStatusClosed})
	requireAppErrorCode(t, err, "forbidden")
	_, err = fixture.ticketService.UpdateStatus(fixture.ctx, fixture.tenantA.ID, ticket.ID, fixture.adminA.ID, fixture.adminA.Role, UpdateTicketStatusInput{Status: "done"})
	requireAppErrorCode(t, err, "invalid_status")
	_, err = fixture.ticketService.UpdateStatus(fixture.ctx, fixture.tenantA.ID, "missing-ticket", fixture.adminA.ID, fixture.adminA.Role, UpdateTicketStatusInput{Status: model.TicketStatusResolved})
	require.Error(t, err)

	require.True(t, isValidPriority(model.TicketPriorityLow))
	require.True(t, isValidPriority(model.TicketPriorityMedium))
	require.True(t, isValidPriority(model.TicketPriorityHigh))
	require.False(t, isValidPriority("urgent"))
	require.True(t, isValidTicketStatus(model.TicketStatusOpen))
	require.True(t, isValidTicketStatus(model.TicketStatusInProgress))
	require.True(t, isValidTicketStatus(model.TicketStatusResolved))
	require.True(t, isValidTicketStatus(model.TicketStatusClosed))
	require.False(t, isValidTicketStatus("done"))
}

func TestConversationAndTicketListUseCache(t *testing.T) {
	fixture := newServiceFixture(t)
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	cacheClient := cache.New(client)
	conversationService := NewConversationService(fixture.store, cacheClient)
	ticketService := NewTicketService(fixture.store, cacheClient)

	conversation, err := conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "cache-1",
		CustomerName:       "Cached Customer",
		Message:            "hello",
	})
	require.NoError(t, err)
	_, err = ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.agentA.ID, EscalateTicketInput{Title: "Cached Ticket", Description: "Investigate"})
	require.NoError(t, err)

	conversations, total, err := conversationService.ListConversations(fixture.ctx, fixture.tenantA.ID, ListConversationsInput{Offset: -1, Limit: 200})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, conversations)
	tickets, total, err := ticketService.ListTickets(fixture.ctx, fixture.tenantA.ID, ListTicketsInput{Offset: -1, Limit: 200})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, tickets)

	sqlDB, err := fixture.store.DB().DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	conversations, total, err = conversationService.ListConversations(fixture.ctx, fixture.tenantA.ID, ListConversationsInput{Offset: -1, Limit: 200})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, conversations)
	tickets, total, err = ticketService.ListTickets(fixture.ctx, fixture.tenantA.ID, ListTicketsInput{Offset: -1, Limit: 200})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, tickets)

	require.Equal(t, 0, normalizeOffset(-1))
	require.Equal(t, 7, normalizeOffset(7))
	require.Equal(t, 20, normalizeLimit(0))
	require.Equal(t, 100, normalizeLimit(101))
	require.Equal(t, 5, normalizeLimit(5))
}

func TestAppendDomainEventBranches(t *testing.T) {
	emptyDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "append-events-empty.db")), &gorm.Config{})
	require.NoError(t, err)

	err = appendDomainEvent(context.Background(), repository.NewStore(emptyDB), "tenant-1", "conversation.created", "conversation", "conversation-1", make(chan int))
	require.Error(t, err)

	err = appendDomainEvent(context.Background(), repository.NewStore(emptyDB), "tenant-1", "conversation.created", "conversation", "conversation-1", map[string]string{"status": "created"})
	require.Error(t, err)

	activityOnlyDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "append-events-activity.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, activityOnlyDB.AutoMigrate(&model.ActivityLog{}))

	err = appendDomainEvent(context.Background(), repository.NewStore(activityOnlyDB), "tenant-1", "conversation.created", "conversation", "conversation-1", map[string]string{"status": "created"})
	require.Error(t, err)
}

func createTenantAgent(t *testing.T, store *repository.Store, tenantID string, id string) model.User {
	t.Helper()

	agent := model.User{
		BaseModel:    model.BaseModel{ID: id},
		TenantID:     tenantID,
		Name:         id,
		Email:        id + "@example.com",
		PasswordHash: "hash",
		Role:         model.RoleAgent,
		Active:       true,
	}
	require.NoError(t, store.DB().Create(&agent).Error)
	return agent
}

func requireAppErrorCode(t *testing.T, err error, code string) {
	t.Helper()

	var appError *apperror.AppError
	require.ErrorAs(t, err, &appError)
	require.Equal(t, code, appError.Code)
}

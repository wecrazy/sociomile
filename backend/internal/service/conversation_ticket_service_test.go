package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"gorm.io/gorm"
)

func TestConversationLifecycleAndAuthorization(t *testing.T) {
	fixture := newServiceFixture(t)

	conversation, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "cust-a-1",
		CustomerName:       "Customer A",
		Message:            "Hello",
	})
	require.NoError(t, err)
	require.Equal(t, model.ConversationStatusOpen, conversation.Status)

	_, err = fixture.conversationService.AssignConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.agentA.ID, fixture.agentA.Role, AssignConversationInput{AgentID: fixture.agentA.ID})
	require.Error(t, err)

	conversation, err = fixture.conversationService.AssignConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.adminA.ID, fixture.adminA.Role, AssignConversationInput{AgentID: fixture.agentA.ID})
	require.NoError(t, err)
	require.NotNil(t, conversation.AssignedAgentID)
	require.Equal(t, fixture.agentA.ID, *conversation.AssignedAgentID)

	_, err = fixture.conversationService.ReplyToConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.agentB.ID, ReplyConversationInput{Message: "Wrong tenant"})
	require.Error(t, err)

	conversation, err = fixture.conversationService.ReplyToConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.agentA.ID, ReplyConversationInput{Message: "We can help."})
	require.NoError(t, err)
	require.Len(t, conversation.Messages, 2)

	ticket, err := fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.agentA.ID, EscalateTicketInput{
		Title:       "Investigate issue",
		Description: "Customer needs deeper follow-up",
		Priority:    model.TicketPriorityHigh,
	})
	require.NoError(t, err)
	require.Equal(t, model.TicketStatusOpen, ticket.Status)

	_, err = fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantA.ID, conversation.ID, fixture.agentA.ID, EscalateTicketInput{
		Title:       "Duplicate escalation",
		Description: "Should fail",
		Priority:    model.TicketPriorityLow,
	})
	require.Error(t, err)

	_, err = fixture.ticketService.UpdateStatus(fixture.ctx, fixture.tenantA.ID, ticket.ID, fixture.agentA.ID, fixture.agentA.Role, UpdateTicketStatusInput{Status: model.TicketStatusResolved})
	require.Error(t, err)

	ticket, err = fixture.ticketService.UpdateStatus(fixture.ctx, fixture.tenantA.ID, ticket.ID, fixture.adminA.ID, fixture.adminA.Role, UpdateTicketStatusInput{Status: model.TicketStatusResolved})
	require.NoError(t, err)
	require.Equal(t, model.TicketStatusResolved, ticket.Status)
}

func TestTenantIsolationForConversationAndTicketQueries(t *testing.T) {
	fixture := newServiceFixture(t)

	conversationA, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantA.ID,
		ChannelKey:         fixture.channelA.Key,
		CustomerExternalID: "cust-a-2",
		CustomerName:       "Tenant A Customer",
		Message:            "Hello from tenant A",
	})
	require.NoError(t, err)

	conversationB, err := fixture.conversationService.HandleWebhook(fixture.ctx, WebhookInput{
		TenantID:           fixture.tenantB.ID,
		ChannelKey:         fixture.channelB.Key,
		CustomerExternalID: "cust-b-1",
		CustomerName:       "Tenant B Customer",
		Message:            "Hello from tenant B",
	})
	require.NoError(t, err)

	conversations, total, err := fixture.conversationService.ListConversations(fixture.ctx, fixture.tenantA.ID, ListConversationsInput{Offset: 0, Limit: 20})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, conversations, 1)
	require.Equal(t, conversationA.ID, conversations[0].ID)

	_, err = fixture.conversationService.GetConversation(fixture.ctx, fixture.tenantA.ID, conversationB.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = fixture.ticketService.EscalateConversation(fixture.ctx, fixture.tenantB.ID, conversationB.ID, fixture.agentB.ID, EscalateTicketInput{
		Title:       "Tenant B issue",
		Description: "Internal follow-up",
		Priority:    model.TicketPriorityMedium,
	})
	require.NoError(t, err)

	tickets, total, err := fixture.ticketService.ListTickets(fixture.ctx, fixture.tenantA.ID, ListTicketsInput{Offset: 0, Limit: 20})
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.Len(t, tickets, 0)

	agents, err := fixture.userService.ListAgents(fixture.ctx, fixture.tenantA.ID)
	require.NoError(t, err)
	require.Len(t, agents, 1)
	require.Equal(t, fixture.agentA.ID, agents[0].ID)
}

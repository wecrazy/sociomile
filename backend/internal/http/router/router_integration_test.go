package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/auth"
	"github.com/wecrazy/sociomile/backend/internal/cache"
	"github.com/wecrazy/sociomile/backend/internal/config"
	"github.com/wecrazy/sociomile/backend/internal/health"
	"github.com/wecrazy/sociomile/backend/internal/http/handler"
	routerpkg "github.com/wecrazy/sociomile/backend/internal/http/router"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"github.com/wecrazy/sociomile/backend/internal/repository"
	"github.com/wecrazy/sociomile/backend/internal/service"
	"github.com/wecrazy/sociomile/backend/seeds"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	tenantAID     = "11111111-1111-1111-1111-111111111111"
	adminAEmail   = "alice.admin@acme.local"
	agentAEmail   = "aaron.agent@acme.local"
	seededAgentID = "11111111-bbbb-bbbb-bbbb-111111111111"
)

type apiFixture struct {
	app *fiber.App
	db  *gorm.DB
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type metaPayload struct {
	Total   int64          `json:"total"`
	Offset  int            `json:"offset"`
	Limit   int            `json:"limit"`
	Filters map[string]any `json:"filters"`
}

type envelope[T any] struct {
	Data  T            `json:"data"`
	Meta  metaPayload  `json:"meta"`
	Error errorPayload `json:"error"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	User        struct {
		Email string `json:"email"`
	} `json:"user"`
}

type conversationResponse struct {
	ID              string  `json:"id"`
	Status          string  `json:"status"`
	AssignedAgentID *string `json:"assigned_agent_id"`
	AssignedAgent   *struct {
		Name string `json:"name"`
	} `json:"assigned_agent"`
	Messages []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"messages"`
	Ticket *struct {
		ID string `json:"id"`
	} `json:"ticket"`
}

type ticketResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type userResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestHealthRootAndMiddleware(t *testing.T) {
	fixture := newAPIFixture(t)

	rootResponse := fixture.request(t, http.MethodGet, "/", nil, map[string]string{
		"X-Request-Id": "req-123",
	})
	defer rootResponse.Body.Close()

	rootBody, err := io.ReadAll(rootResponse.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rootResponse.StatusCode)
	require.Equal(t, "req-123", rootResponse.Header.Get("X-Request-Id"))
	require.Contains(t, string(rootBody), "Sociomile API listening on 8080")

	healthResponse := fixture.request(t, http.MethodGet, "/health", nil, nil)
	defer healthResponse.Body.Close()

	var healthBody map[string]any
	decodeResponse(t, healthResponse, &healthBody)
	require.Equal(t, http.StatusOK, healthResponse.StatusCode)
	require.Equal(t, "ok", healthBody["status"])
	services, ok := healthBody["services"].(map[string]any)
	require.True(t, ok)
	apiService, ok := services["api"].(map[string]any)
	require.True(t, ok)
	workerService, ok := services["worker"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "online", apiService["status"])
	require.Equal(t, "unknown", workerService["status"])
	require.NotEmpty(t, healthResponse.Header.Get("X-Request-Id"))

	protectedResponse := fixture.request(t, http.MethodGet, "/api/v1/conversations", nil, nil)
	defer protectedResponse.Body.Close()

	var protectedBody envelope[any]
	decodeResponse(t, protectedResponse, &protectedBody)
	require.Equal(t, http.StatusUnauthorized, protectedResponse.StatusCode)
	require.Equal(t, "missing_token", protectedBody.Error.Code)

	optionsRequest := httptest.NewRequest(http.MethodOptions, "/api/v1/conversations", nil)
	optionsResponse, err := fixture.app.Test(optionsRequest)
	require.NoError(t, err)
	defer optionsResponse.Body.Close()

	require.Equal(t, http.StatusNoContent, optionsResponse.StatusCode)
	require.Equal(t, "*", optionsResponse.Header.Get("Access-Control-Allow-Origin"))
	require.Contains(t, optionsResponse.Header.Get("Access-Control-Allow-Methods"), http.MethodPatch)
}

func TestAuthenticationAndProtectedRoutes(t *testing.T) {
	fixture := newAPIFixture(t)

	missingFieldResponse := fixture.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": adminAEmail,
	}, nil)
	defer missingFieldResponse.Body.Close()

	var missingFieldBody envelope[any]
	decodeResponse(t, missingFieldResponse, &missingFieldBody)
	require.Equal(t, http.StatusBadRequest, missingFieldResponse.StatusCode)
	require.Equal(t, "invalid_request", missingFieldBody.Error.Code)

	adminToken := fixture.login(t, adminAEmail, "Password123!")

	meResponse := fixture.request(t, http.MethodGet, "/api/v1/auth/me", nil, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer meResponse.Body.Close()

	var meBody envelope[struct {
		Email string `json:"email"`
	}]
	decodeResponse(t, meResponse, &meBody)
	require.Equal(t, http.StatusOK, meResponse.StatusCode)
	require.Equal(t, adminAEmail, meBody.Data.Email)

	invalidHeaderResponse := fixture.request(t, http.MethodGet, "/api/v1/users/agents", nil, map[string]string{
		"Authorization": "Token nope",
	})
	defer invalidHeaderResponse.Body.Close()

	var invalidHeaderBody envelope[any]
	decodeResponse(t, invalidHeaderResponse, &invalidHeaderBody)
	require.Equal(t, http.StatusUnauthorized, invalidHeaderResponse.StatusCode)
	require.Equal(t, "invalid_token", invalidHeaderBody.Error.Code)

	invalidTokenResponse := fixture.request(t, http.MethodGet, "/api/v1/users/agents", nil, map[string]string{
		"Authorization": "Bearer definitely-not-a-jwt",
	})
	defer invalidTokenResponse.Body.Close()

	var invalidTokenBody envelope[any]
	decodeResponse(t, invalidTokenResponse, &invalidTokenBody)
	require.Equal(t, http.StatusUnauthorized, invalidTokenResponse.StatusCode)
	require.Equal(t, "invalid_token", invalidTokenBody.Error.Code)

	agentsResponse := fixture.request(t, http.MethodGet, "/api/v1/users/agents", nil, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer agentsResponse.Body.Close()

	var agentsBody envelope[[]userResponse]
	decodeResponse(t, agentsResponse, &agentsBody)
	require.Equal(t, http.StatusOK, agentsResponse.StatusCode)
	require.Len(t, agentsBody.Data, 1)
	require.Equal(t, seededAgentID, agentsBody.Data[0].ID)

	loginEnvelope := fixture.loginEnvelope(t, agentAEmail, "Password123!")
	require.Equal(t, agentAEmail, loginEnvelope.Data.User.Email)
	require.NotEmpty(t, loginEnvelope.Data.AccessToken)
}

func TestConversationAndTicketHTTPFlow(t *testing.T) {
	fixture := newAPIFixture(t)
	adminToken := fixture.login(t, adminAEmail, "Password123!")
	agentToken := fixture.login(t, agentAEmail, "Password123!")

	conversationA := fixture.createConversation(t, "cust-http-1", "Need assignment and close")

	conversationListResponse := fixture.request(t, http.MethodGet, "/api/v1/conversations?offset=bad&limit=bad&status=open", nil, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer conversationListResponse.Body.Close()

	var conversationListBody envelope[[]conversationResponse]
	decodeResponse(t, conversationListResponse, &conversationListBody)
	require.Equal(t, http.StatusOK, conversationListResponse.StatusCode)
	require.Equal(t, 0, conversationListBody.Meta.Offset)
	require.Equal(t, 20, conversationListBody.Meta.Limit)
	require.GreaterOrEqual(t, conversationListBody.Meta.Total, int64(2))
	require.Equal(t, "open", conversationListBody.Meta.Filters["status"])

	assignResponse := fixture.request(t, http.MethodPatch, "/api/v1/conversations/"+conversationA.ID+"/assign", map[string]string{
		"agent_id": seededAgentID,
	}, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer assignResponse.Body.Close()

	var assignBody envelope[conversationResponse]
	decodeResponse(t, assignResponse, &assignBody)
	require.Equal(t, http.StatusOK, assignResponse.StatusCode)
	require.NotNil(t, assignBody.Data.AssignedAgentID)
	require.Equal(t, seededAgentID, *assignBody.Data.AssignedAgentID)

	detailResponse := fixture.request(t, http.MethodGet, "/api/v1/conversations/"+conversationA.ID, nil, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer detailResponse.Body.Close()

	var detailBody envelope[conversationResponse]
	decodeResponse(t, detailResponse, &detailBody)
	require.Equal(t, http.StatusOK, detailResponse.StatusCode)
	require.NotNil(t, detailBody.Data.AssignedAgent)
	require.Equal(t, "Aaron Agent", detailBody.Data.AssignedAgent.Name)

	forbiddenReplyResponse := fixture.request(t, http.MethodPost, "/api/v1/conversations/"+conversationA.ID+"/messages", map[string]string{
		"message": "Admin should not reply here",
	}, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer forbiddenReplyResponse.Body.Close()

	var forbiddenReplyBody envelope[any]
	decodeResponse(t, forbiddenReplyResponse, &forbiddenReplyBody)
	require.Equal(t, http.StatusForbidden, forbiddenReplyResponse.StatusCode)
	require.Equal(t, "forbidden", forbiddenReplyBody.Error.Code)

	replyResponse := fixture.request(t, http.MethodPost, "/api/v1/conversations/"+conversationA.ID+"/messages", map[string]string{
		"message": "I am on it.",
	}, map[string]string{
		"Authorization": "Bearer " + agentToken,
	})
	defer replyResponse.Body.Close()

	var replyBody envelope[conversationResponse]
	decodeResponse(t, replyResponse, &replyBody)
	require.Equal(t, http.StatusOK, replyResponse.StatusCode)
	require.Len(t, replyBody.Data.Messages, 2)

	closeResponse := fixture.request(t, http.MethodPatch, "/api/v1/conversations/"+conversationA.ID+"/close", nil, map[string]string{
		"Authorization": "Bearer " + agentToken,
	})
	defer closeResponse.Body.Close()

	var closeBody envelope[conversationResponse]
	decodeResponse(t, closeResponse, &closeBody)
	require.Equal(t, http.StatusOK, closeResponse.StatusCode)
	require.Equal(t, model.ConversationStatusClosed, closeBody.Data.Status)

	conversationB := fixture.createConversation(t, "cust-http-2", "Need escalation")

	escalateResponse := fixture.request(t, http.MethodPost, "/api/v1/conversations/"+conversationB.ID+"/escalate", map[string]string{
		"title":       "Escalated via HTTP",
		"description": "Customer needs specialist support",
		"priority":    model.TicketPriorityHigh,
	}, map[string]string{
		"Authorization": "Bearer " + agentToken,
	})
	defer escalateResponse.Body.Close()

	var escalateBody envelope[ticketResponse]
	decodeResponse(t, escalateResponse, &escalateBody)
	require.Equal(t, http.StatusCreated, escalateResponse.StatusCode)
	require.NotEmpty(t, escalateBody.Data.ID)

	ticketListResponse := fixture.request(t, http.MethodGet, "/api/v1/tickets?offset=0&limit=10&status=open", nil, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer ticketListResponse.Body.Close()

	var ticketListBody envelope[[]ticketResponse]
	decodeResponse(t, ticketListResponse, &ticketListBody)
	require.Equal(t, http.StatusOK, ticketListResponse.StatusCode)
	require.GreaterOrEqual(t, ticketListBody.Meta.Total, int64(1))
	require.Equal(t, "open", ticketListBody.Meta.Filters["status"])

	ticketDetailResponse := fixture.request(t, http.MethodGet, "/api/v1/tickets/"+escalateBody.Data.ID, nil, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer ticketDetailResponse.Body.Close()

	var ticketDetailBody envelope[ticketResponse]
	decodeResponse(t, ticketDetailResponse, &ticketDetailBody)
	require.Equal(t, http.StatusOK, ticketDetailResponse.StatusCode)
	require.Equal(t, escalateBody.Data.ID, ticketDetailBody.Data.ID)

	forbiddenStatusResponse := fixture.request(t, http.MethodPatch, "/api/v1/tickets/"+escalateBody.Data.ID+"/status", map[string]string{
		"status": model.TicketStatusResolved,
	}, map[string]string{
		"Authorization": "Bearer " + agentToken,
	})
	defer forbiddenStatusResponse.Body.Close()

	var forbiddenStatusBody envelope[any]
	decodeResponse(t, forbiddenStatusResponse, &forbiddenStatusBody)
	require.Equal(t, http.StatusForbidden, forbiddenStatusResponse.StatusCode)
	require.Equal(t, "forbidden", forbiddenStatusBody.Error.Code)

	statusResponse := fixture.request(t, http.MethodPatch, "/api/v1/tickets/"+escalateBody.Data.ID+"/status", map[string]string{
		"status": model.TicketStatusResolved,
	}, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer statusResponse.Body.Close()

	var statusBody envelope[ticketResponse]
	decodeResponse(t, statusResponse, &statusBody)
	require.Equal(t, http.StatusOK, statusResponse.StatusCode)
	require.Equal(t, model.TicketStatusResolved, statusBody.Data.Status)
}

func TestHandlerValidationAndLookupFailures(t *testing.T) {
	fixture := newAPIFixture(t)
	adminToken := fixture.login(t, adminAEmail, "Password123!")
	agentToken := fixture.login(t, agentAEmail, "Password123!")

	missingUserToken, err := auth.GenerateToken("test-secret", time.Minute, &model.User{
		BaseModel: model.BaseModel{ID: "missing-user"},
		TenantID:  tenantAID,
		Email:     "missing@example.com",
		Role:      model.RoleAdmin,
	})
	require.NoError(t, err)

	invalidJSONCases := []struct {
		name    string
		method  string
		path    string
		headers map[string]string
	}{
		{name: "login", method: http.MethodPost, path: "/api/v1/auth/login"},
		{name: "channel webhook", method: http.MethodPost, path: "/api/v1/channel/webhook"},
		{name: "assign", method: http.MethodPatch, path: "/api/v1/conversations/conversation-1/assign", headers: map[string]string{"Authorization": "Bearer " + adminToken}},
		{name: "reply", method: http.MethodPost, path: "/api/v1/conversations/conversation-1/messages", headers: map[string]string{"Authorization": "Bearer " + agentToken}},
		{name: "escalate", method: http.MethodPost, path: "/api/v1/conversations/conversation-1/escalate", headers: map[string]string{"Authorization": "Bearer " + agentToken}},
		{name: "ticket status", method: http.MethodPatch, path: "/api/v1/tickets/ticket-1/status", headers: map[string]string{"Authorization": "Bearer " + adminToken}},
	}

	for _, testCase := range invalidJSONCases {
		t.Run(testCase.name+" invalid json", func(t *testing.T) {
			response := fixture.rawRequest(t, testCase.method, testCase.path, []byte("{"), testCase.headers)
			defer response.Body.Close()

			var body envelope[any]
			decodeResponse(t, response, &body)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			require.Equal(t, "invalid_request", body.Error.Code)
		})
	}

	validationCases := []struct {
		name         string
		method       string
		path         string
		body         any
		headers      map[string]string
		expectedCode string
	}{
		{
			name:         "invalid webhook payload",
			method:       http.MethodPost,
			path:         "/api/v1/channel/webhook",
			body:         map[string]string{},
			expectedCode: "invalid_tenant",
		},
		{
			name:         "invalid assignment",
			method:       http.MethodPatch,
			path:         "/api/v1/conversations/conversation-1/assign",
			body:         map[string]string{"agent_id": "   "},
			headers:      map[string]string{"Authorization": "Bearer " + adminToken},
			expectedCode: "invalid_assignment",
		},
		{
			name:         "invalid reply message",
			method:       http.MethodPost,
			path:         "/api/v1/conversations/conversation-1/messages",
			body:         map[string]string{"message": "   "},
			headers:      map[string]string{"Authorization": "Bearer " + agentToken},
			expectedCode: "invalid_message",
		},
		{
			name:         "invalid priority",
			method:       http.MethodPost,
			path:         "/api/v1/conversations/conversation-1/escalate",
			body:         map[string]string{"title": "Need help", "description": "Escalate this", "priority": "urgent"},
			headers:      map[string]string{"Authorization": "Bearer " + agentToken},
			expectedCode: "invalid_priority",
		},
		{
			name:         "invalid ticket status",
			method:       http.MethodPatch,
			path:         "/api/v1/tickets/ticket-1/status",
			body:         map[string]string{"status": "bad-status"},
			headers:      map[string]string{"Authorization": "Bearer " + adminToken},
			expectedCode: "invalid_status",
		},
	}

	for _, testCase := range validationCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := fixture.request(t, testCase.method, testCase.path, testCase.body, testCase.headers)
			defer response.Body.Close()

			var body envelope[any]
			decodeResponse(t, response, &body)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			require.Equal(t, testCase.expectedCode, body.Error.Code)
		})
	}

	notFoundCases := []struct {
		name    string
		method  string
		path    string
		body    any
		headers map[string]string
	}{
		{name: "missing me user", method: http.MethodGet, path: "/api/v1/auth/me", headers: map[string]string{"Authorization": "Bearer " + missingUserToken}},
		{name: "missing conversation detail", method: http.MethodGet, path: "/api/v1/conversations/missing", headers: map[string]string{"Authorization": "Bearer " + adminToken}},
		{name: "missing conversation close", method: http.MethodPatch, path: "/api/v1/conversations/missing/close", headers: map[string]string{"Authorization": "Bearer " + adminToken}},
		{name: "missing ticket detail", method: http.MethodGet, path: "/api/v1/tickets/missing", headers: map[string]string{"Authorization": "Bearer " + adminToken}},
	}

	for _, testCase := range notFoundCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := fixture.request(t, testCase.method, testCase.path, testCase.body, testCase.headers)
			defer response.Body.Close()

			var body envelope[any]
			decodeResponse(t, response, &body)
			require.Equal(t, http.StatusNotFound, response.StatusCode)
			require.Equal(t, "not_found", body.Error.Code)
		})
	}
}

func TestHandlersReturnInternalErrorsWhenDatabaseIsClosed(t *testing.T) {
	fixture := newAPIFixture(t)
	adminToken := fixture.login(t, adminAEmail, "Password123!")

	sqlDB, err := fixture.db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	testCases := []struct {
		name   string
		path   string
		method string
	}{
		{name: "me", method: http.MethodGet, path: "/api/v1/auth/me"},
		{name: "agents", method: http.MethodGet, path: "/api/v1/users/agents"},
		{name: "conversations", method: http.MethodGet, path: "/api/v1/conversations"},
		{name: "tickets", method: http.MethodGet, path: "/api/v1/tickets"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := fixture.request(t, testCase.method, testCase.path, nil, map[string]string{
				"Authorization": "Bearer " + adminToken,
			})
			defer response.Body.Close()

			var body envelope[any]
			decodeResponse(t, response, &body)
			require.Equal(t, http.StatusInternalServerError, response.StatusCode)
			require.Equal(t, "internal_error", body.Error.Code)
		})
	}
}

func TestConversationAndTicketConflictFlows(t *testing.T) {
	fixture := newAPIFixture(t)
	adminToken := fixture.login(t, adminAEmail, "Password123!")
	agentToken := fixture.login(t, agentAEmail, "Password123!")

	invalidChannelResponse := fixture.request(t, http.MethodPost, "/api/v1/channel/webhook", map[string]string{
		"tenant_id":            tenantAID,
		"channel_key":          "telegram",
		"customer_external_id": "cust-invalid-channel",
		"message":              "hello",
	}, nil)
	defer invalidChannelResponse.Body.Close()

	var invalidChannelBody envelope[any]
	decodeResponse(t, invalidChannelResponse, &invalidChannelBody)
	require.Equal(t, http.StatusBadRequest, invalidChannelResponse.StatusCode)
	require.Equal(t, "invalid_channel", invalidChannelBody.Error.Code)

	missingTenantResponse := fixture.request(t, http.MethodPost, "/api/v1/channel/webhook", map[string]string{
		"tenant_id":            "missing-tenant",
		"channel_key":          "whatsapp",
		"customer_external_id": "cust-missing-tenant",
		"message":              "hello",
	}, nil)
	defer missingTenantResponse.Body.Close()

	var missingTenantBody envelope[any]
	decodeResponse(t, missingTenantResponse, &missingTenantBody)
	require.Equal(t, http.StatusNotFound, missingTenantResponse.StatusCode)
	require.Equal(t, "tenant_not_found", missingTenantBody.Error.Code)

	duplicateEscalationConversation := fixture.createConversation(t, "cust-ticket-exists", "Need escalation twice")
	firstEscalationResponse := fixture.request(t, http.MethodPost, "/api/v1/conversations/"+duplicateEscalationConversation.ID+"/escalate", map[string]string{
		"title":       "Escalate once",
		"description": "First ticket creation",
		"priority":    model.TicketPriorityHigh,
	}, map[string]string{
		"Authorization": "Bearer " + agentToken,
	})
	defer firstEscalationResponse.Body.Close()

	var firstEscalationBody envelope[ticketResponse]
	decodeResponse(t, firstEscalationResponse, &firstEscalationBody)
	require.Equal(t, http.StatusCreated, firstEscalationResponse.StatusCode)
	require.NotEmpty(t, firstEscalationBody.Data.ID)

	duplicateEscalationResponse := fixture.request(t, http.MethodPost, "/api/v1/conversations/"+duplicateEscalationConversation.ID+"/escalate", map[string]string{
		"title":       "Escalate again",
		"description": "Should fail",
		"priority":    model.TicketPriorityHigh,
	}, map[string]string{
		"Authorization": "Bearer " + agentToken,
	})
	defer duplicateEscalationResponse.Body.Close()

	var duplicateEscalationBody envelope[any]
	decodeResponse(t, duplicateEscalationResponse, &duplicateEscalationBody)
	require.Equal(t, http.StatusConflict, duplicateEscalationResponse.StatusCode)
	require.Equal(t, "ticket_exists", duplicateEscalationBody.Error.Code)

	closedConversation := fixture.createConversation(t, "cust-closed-escalation", "Close before escalate")
	assignResponse := fixture.request(t, http.MethodPatch, "/api/v1/conversations/"+closedConversation.ID+"/assign", map[string]string{
		"agent_id": seededAgentID,
	}, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer assignResponse.Body.Close()
	require.Equal(t, http.StatusOK, assignResponse.StatusCode)

	closeResponse := fixture.request(t, http.MethodPatch, "/api/v1/conversations/"+closedConversation.ID+"/close", nil, map[string]string{
		"Authorization": "Bearer " + agentToken,
	})
	defer closeResponse.Body.Close()
	require.Equal(t, http.StatusOK, closeResponse.StatusCode)

	replyClosedResponse := fixture.request(t, http.MethodPost, "/api/v1/conversations/"+closedConversation.ID+"/messages", map[string]string{
		"message": "Too late",
	}, map[string]string{
		"Authorization": "Bearer " + agentToken,
	})
	defer replyClosedResponse.Body.Close()

	var replyClosedBody envelope[any]
	decodeResponse(t, replyClosedResponse, &replyClosedBody)
	require.Equal(t, http.StatusConflict, replyClosedResponse.StatusCode)
	require.Equal(t, "conversation_closed", replyClosedBody.Error.Code)

	escalateClosedResponse := fixture.request(t, http.MethodPost, "/api/v1/conversations/"+closedConversation.ID+"/escalate", map[string]string{
		"title":       "Escalate closed conversation",
		"description": "Should fail",
		"priority":    model.TicketPriorityMedium,
	}, map[string]string{
		"Authorization": "Bearer " + agentToken,
	})
	defer escalateClosedResponse.Body.Close()

	var escalateClosedBody envelope[any]
	decodeResponse(t, escalateClosedResponse, &escalateClosedBody)
	require.Equal(t, http.StatusConflict, escalateClosedResponse.StatusCode)
	require.Equal(t, "conversation_closed", escalateClosedBody.Error.Code)

	missingTicketStatusResponse := fixture.request(t, http.MethodPatch, "/api/v1/tickets/missing/status", map[string]string{
		"status": model.TicketStatusResolved,
	}, map[string]string{
		"Authorization": "Bearer " + adminToken,
	})
	defer missingTicketStatusResponse.Body.Close()

	var missingTicketStatusBody envelope[any]
	decodeResponse(t, missingTicketStatusResponse, &missingTicketStatusBody)
	require.Equal(t, http.StatusNotFound, missingTicketStatusResponse.StatusCode)
	require.Equal(t, "not_found", missingTicketStatusBody.Error.Code)
}

func newAPIFixture(t *testing.T) *apiFixture {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "router-test.db")), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.Tenant{},
		&model.User{},
		&model.Channel{},
		&model.Customer{},
		&model.Conversation{},
		&model.Message{},
		&model.Ticket{},
		&model.ActivityLog{},
		&model.OutboxEvent{},
	)
	require.NoError(t, err)
	require.NoError(t, seeds.LoadDemoData(context.Background(), db))

	store := repository.NewStore(db)
	cacheClient := cache.New(nil)

	authService := service.NewAuthService(store, "test-secret", 15*time.Minute)
	userService := service.NewUserService(store)
	conversationService := service.NewConversationService(store, cacheClient)
	ticketService := service.NewTicketService(store, cacheClient)

	swaggerFile := filepath.Join(t.TempDir(), "openapi.yaml")
	require.NoError(t, os.WriteFile(swaggerFile, []byte("openapi: 3.0.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}\n"), 0o644))

	app := routerpkg.New(
		config.Config{
			Port:           8080,
			JWTSecret:      "test-secret",
			AccessTokenTTL: 15 * time.Minute,
			SwaggerFile:    swaggerFile,
		},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		health.NewReporter(cacheClient),
		handler.NewAuthHandler(authService),
		handler.NewUserHandler(userService),
		handler.NewConversationHandler(conversationService),
		handler.NewTicketHandler(ticketService),
	)

	t.Cleanup(func() {
		_ = app.Shutdown()
	})

	return &apiFixture{app: app, db: db}
}

func (f *apiFixture) login(t *testing.T, email string, password string) string {
	t.Helper()
	response := f.loginEnvelope(t, email, password)
	return response.Data.AccessToken
}

func (f *apiFixture) loginEnvelope(t *testing.T, email string, password string) envelope[loginResponse] {
	t.Helper()
	response := f.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, nil)
	defer response.Body.Close()

	var body envelope[loginResponse]
	decodeResponse(t, response, &body)
	require.Equal(t, http.StatusOK, response.StatusCode)
	return body
}

func (f *apiFixture) createConversation(t *testing.T, externalID string, message string) conversationResponse {
	t.Helper()
	response := f.request(t, http.MethodPost, "/api/v1/channel/webhook", map[string]string{
		"tenant_id":            tenantAID,
		"channel_key":          "whatsapp",
		"customer_external_id": externalID,
		"customer_name":        "HTTP Test Customer",
		"message":              message,
	}, nil)
	defer response.Body.Close()

	var body envelope[conversationResponse]
	decodeResponse(t, response, &body)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	return body.Data
}

func (f *apiFixture) request(t *testing.T, method string, path string, body any, headers map[string]string) *http.Response {
	t.Helper()

	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		payload = bytes.NewReader(encoded)
	}

	request := httptest.NewRequest(method, path, payload)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := f.app.Test(request)
	require.NoError(t, err)
	return response
}

func (f *apiFixture) rawRequest(t *testing.T, method string, path string, body []byte, headers map[string]string) *http.Response {
	t.Helper()

	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := f.app.Test(request)
	require.NoError(t, err)
	return response
}

func decodeResponse[T any](t *testing.T, response *http.Response, target *T) {
	t.Helper()
	require.NoError(t, json.NewDecoder(response.Body).Decode(target))
}

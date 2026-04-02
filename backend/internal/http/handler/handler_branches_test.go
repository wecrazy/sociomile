package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/auth"
	"github.com/wecrazy/sociomile/backend/internal/http/handler"
	"github.com/wecrazy/sociomile/backend/internal/http/middleware"
	"github.com/wecrazy/sociomile/backend/internal/model"
)

const handlerTestSecret = "test-secret"

type handlerErrorEnvelope struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func TestHandlersRequireClaims(t *testing.T) {
	authHandler := handler.NewAuthHandler(nil)
	conversationHandler := handler.NewConversationHandler(nil)
	ticketHandler := handler.NewTicketHandler(nil)
	userHandler := handler.NewUserHandler(nil)

	testCases := []struct {
		name     string
		method   string
		path     string
		register func(*fiber.App)
	}{
		{
			name:   "auth me",
			method: http.MethodGet,
			path:   "/auth/me",
			register: func(app *fiber.App) {
				app.Get("/auth/me", authHandler.Me)
			},
		},
		{
			name:   "user agents",
			method: http.MethodGet,
			path:   "/users/agents",
			register: func(app *fiber.App) {
				app.Get("/users/agents", userHandler.ListAgents)
			},
		},
		{
			name:   "conversation list",
			method: http.MethodGet,
			path:   "/conversations",
			register: func(app *fiber.App) {
				app.Get("/conversations", conversationHandler.List)
			},
		},
		{
			name:   "conversation detail",
			method: http.MethodGet,
			path:   "/conversations/conversation-1",
			register: func(app *fiber.App) {
				app.Get("/conversations/:id", conversationHandler.Detail)
			},
		},
		{
			name:   "conversation assign",
			method: http.MethodPatch,
			path:   "/conversations/conversation-1/assign",
			register: func(app *fiber.App) {
				app.Patch("/conversations/:id/assign", conversationHandler.Assign)
			},
		},
		{
			name:   "conversation reply",
			method: http.MethodPost,
			path:   "/conversations/conversation-1/messages",
			register: func(app *fiber.App) {
				app.Post("/conversations/:id/messages", conversationHandler.Reply)
			},
		},
		{
			name:   "conversation close",
			method: http.MethodPatch,
			path:   "/conversations/conversation-1/close",
			register: func(app *fiber.App) {
				app.Patch("/conversations/:id/close", conversationHandler.Close)
			},
		},
		{
			name:   "ticket list",
			method: http.MethodGet,
			path:   "/tickets",
			register: func(app *fiber.App) {
				app.Get("/tickets", ticketHandler.List)
			},
		},
		{
			name:   "ticket detail",
			method: http.MethodGet,
			path:   "/tickets/ticket-1",
			register: func(app *fiber.App) {
				app.Get("/tickets/:id", ticketHandler.Detail)
			},
		},
		{
			name:   "ticket escalate",
			method: http.MethodPost,
			path:   "/conversations/conversation-1/escalate",
			register: func(app *fiber.App) {
				app.Post("/conversations/:id/escalate", ticketHandler.Escalate)
			},
		},
		{
			name:   "ticket status",
			method: http.MethodPatch,
			path:   "/tickets/ticket-1/status",
			register: func(app *fiber.App) {
				app.Patch("/tickets/:id/status", ticketHandler.UpdateStatus)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			app := fiber.New()
			testCase.register(app)

			response := performRawRequest(t, app, testCase.method, testCase.path, []byte("{}"), nil)
			defer response.Body.Close()

			var body handlerErrorEnvelope
			decodeHandlerResponse(t, response, &body)
			require.Equal(t, http.StatusUnauthorized, response.StatusCode)
			require.Equal(t, "missing_claims", body.Error.Code)
		})
	}
}

func TestHandlersRejectInvalidJSON(t *testing.T) {
	authHandler := handler.NewAuthHandler(nil)
	conversationHandler := handler.NewConversationHandler(nil)
	ticketHandler := handler.NewTicketHandler(nil)
	token := testHandlerToken(t, model.RoleAdmin)

	testCases := []struct {
		name     string
		method   string
		path     string
		useAuth  bool
		register func(*fiber.App)
	}{
		{
			name:    "login",
			method:  http.MethodPost,
			path:    "/auth/login",
			useAuth: false,
			register: func(app *fiber.App) {
				app.Post("/auth/login", authHandler.Login)
			},
		},
		{
			name:    "channel webhook",
			method:  http.MethodPost,
			path:    "/channel/webhook",
			useAuth: false,
			register: func(app *fiber.App) {
				app.Post("/channel/webhook", conversationHandler.ChannelWebhook)
			},
		},
		{
			name:    "conversation assign",
			method:  http.MethodPatch,
			path:    "/conversations/conversation-1/assign",
			useAuth: true,
			register: func(app *fiber.App) {
				app.Patch("/conversations/:id/assign", conversationHandler.Assign)
			},
		},
		{
			name:    "conversation reply",
			method:  http.MethodPost,
			path:    "/conversations/conversation-1/messages",
			useAuth: true,
			register: func(app *fiber.App) {
				app.Post("/conversations/:id/messages", conversationHandler.Reply)
			},
		},
		{
			name:    "ticket escalate",
			method:  http.MethodPost,
			path:    "/conversations/conversation-1/escalate",
			useAuth: true,
			register: func(app *fiber.App) {
				app.Post("/conversations/:id/escalate", ticketHandler.Escalate)
			},
		},
		{
			name:    "ticket status",
			method:  http.MethodPatch,
			path:    "/tickets/ticket-1/status",
			useAuth: true,
			register: func(app *fiber.App) {
				app.Patch("/tickets/:id/status", ticketHandler.UpdateStatus)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			app := fiber.New()
			if testCase.useAuth {
				app.Use(middleware.Auth(handlerTestSecret))
			}
			testCase.register(app)

			headers := map[string]string{}
			if testCase.useAuth {
				headers["Authorization"] = "Bearer " + token
			}

			response := performRawRequest(t, app, testCase.method, testCase.path, []byte("{"), headers)
			defer response.Body.Close()

			var body handlerErrorEnvelope
			decodeHandlerResponse(t, response, &body)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			require.Equal(t, "invalid_request", body.Error.Code)
		})
	}
}

func performRawRequest(t *testing.T, app *fiber.App, method string, path string, body []byte, headers map[string]string) *http.Response {
	t.Helper()

	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := app.Test(request)
	require.NoError(t, err)
	return response
}

func decodeHandlerResponse(t *testing.T, response *http.Response, target any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(response.Body).Decode(target))
}

func testHandlerToken(t *testing.T, role string) string {
	t.Helper()

	token, err := auth.GenerateToken(handlerTestSecret, time.Minute, &model.User{
		BaseModel: model.BaseModel{ID: "handler-user"},
		TenantID:  "handler-tenant",
		Email:     "handler@example.com",
		Role:      role,
	})
	require.NoError(t, err)
	return token
}

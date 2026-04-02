package router

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/contrib/v3/swaggerui"
	"github.com/gofiber/fiber/v3"
	"github.com/wecrazy/sociomile/backend/internal/config"
	"github.com/wecrazy/sociomile/backend/internal/health"
	"github.com/wecrazy/sociomile/backend/internal/http/handler"
	"github.com/wecrazy/sociomile/backend/internal/http/middleware"
	"github.com/wecrazy/sociomile/backend/internal/http/response"
	"github.com/wecrazy/sociomile/backend/internal/model"
)

// New wires the Fiber application with middleware, routes, and Swagger.
func New(cfg config.Config, logger *slog.Logger, healthReporter *health.Reporter, authHandler *handler.AuthHandler, userHandler *handler.UserHandler, conversationHandler *handler.ConversationHandler, ticketHandler *handler.TicketHandler) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			logger.Error("request failed",
				slog.String("request_id", middleware.CurrentRequestID(c)),
				slog.String("path", c.Path()),
				slog.String("method", c.Method()),
				slog.Any("error", err),
			)
			return response.Error(c, err)
		},
	})

	app.Use(middleware.RequestID())
	app.Use(middleware.CORS())
	app.Use(func(c fiber.Ctx) error {
		startedAt := time.Now()
		err := c.Next()
		logger.Info("request completed",
			slog.String("request_id", middleware.CurrentRequestID(c)),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", c.Response().StatusCode()),
			slog.Duration("duration", time.Since(startedAt)),
		)
		return err
	})

	app.Get("/health", func(c fiber.Ctx) error {
		if healthReporter == nil {
			return c.JSON(health.Payload{
				Status: "ok",
				Port:   cfg.Port,
				Services: map[string]health.Service{
					"api":    {Status: health.ServiceStatusOnline},
					"worker": {Status: health.ServiceStatusUnknown},
				},
			})
		}

		return c.JSON(healthReporter.Snapshot(c.Context(), cfg.Port))
	})

	app.Use(swaggerui.New(swaggerui.Config{
		BasePath: "/",
		FilePath: cfg.SwaggerFile,
		Path:     "swagger",
		Title:    "Sociomile API Docs",
	}))

	api := app.Group("/api")
	v1 := api.Group("/v1")

	v1.Post("/auth/login", authHandler.Login)
	v1.Post("/channel/webhook", conversationHandler.ChannelWebhook)

	protected := v1.Group("/", middleware.Auth(cfg.JWTSecret))
	protected.Get("/auth/me", authHandler.Me)
	protected.Get("/users/agents", userHandler.ListAgents)

	conversations := protected.Group("/conversations")
	conversations.Get("/", conversationHandler.List)
	conversations.Get("/:id", conversationHandler.Detail)
	conversations.Post("/:id/messages", middleware.RequireRoles(model.RoleAgent), conversationHandler.Reply)
	conversations.Patch("/:id/assign", middleware.RequireRoles(model.RoleAdmin), conversationHandler.Assign)
	conversations.Patch("/:id/close", middleware.RequireRoles(model.RoleAdmin, model.RoleAgent), conversationHandler.Close)
	conversations.Post("/:id/escalate", middleware.RequireRoles(model.RoleAgent), ticketHandler.Escalate)

	tickets := protected.Group("/tickets")
	tickets.Get("/", ticketHandler.List)
	tickets.Get("/:id", ticketHandler.Detail)
	tickets.Patch("/:id/status", middleware.RequireRoles(model.RoleAdmin), ticketHandler.UpdateStatus)

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("Sociomile API listening on %d", cfg.Port))
	})

	return app
}

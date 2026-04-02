package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/auth"
	"github.com/wecrazy/sociomile/backend/internal/http/response"
	"github.com/wecrazy/sociomile/backend/internal/model"
)

type middlewareErrorEnvelope struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func TestRequireRolesBranches(t *testing.T) {
	t.Run("missing claims", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ErrorHandler: func(c fiber.Ctx, err error) error {
				return response.Error(c, err)
			},
		})
		app.Use(RequireRoles(model.RoleAdmin))
		app.Get("/", func(c fiber.Ctx) error {
			return c.SendStatus(http.StatusNoContent)
		})

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		response, err := app.Test(request)
		require.NoError(t, err)
		defer response.Body.Close()

		var body middlewareErrorEnvelope
		require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
		require.Equal(t, http.StatusUnauthorized, response.StatusCode)
		require.Equal(t, "missing_claims", body.Error.Code)
	})

	t.Run("forbidden role", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ErrorHandler: func(c fiber.Ctx, err error) error {
				return response.Error(c, err)
			},
		})
		app.Use(func(c fiber.Ctx) error {
			c.Locals(claimsKey, &auth.Claims{Role: model.RoleAgent})
			return c.Next()
		})
		app.Use(RequireRoles(model.RoleAdmin))
		app.Get("/", func(c fiber.Ctx) error {
			return c.SendStatus(http.StatusNoContent)
		})

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		response, err := app.Test(request)
		require.NoError(t, err)
		defer response.Body.Close()

		var body middlewareErrorEnvelope
		require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
		require.Equal(t, http.StatusForbidden, response.StatusCode)
		require.Equal(t, "forbidden", body.Error.Code)
	})

	t.Run("allowed role", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ErrorHandler: func(c fiber.Ctx, err error) error {
				return response.Error(c, err)
			},
		})
		app.Use(func(c fiber.Ctx) error {
			c.Locals(claimsKey, &auth.Claims{Role: model.RoleAdmin})
			return c.Next()
		})
		app.Use(RequireRoles(model.RoleAdmin, model.RoleAgent))
		app.Get("/", func(c fiber.Ctx) error {
			return c.SendStatus(http.StatusNoContent)
		})

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		response, err := app.Test(request)
		require.NoError(t, err)
		defer response.Body.Close()

		require.Equal(t, http.StatusNoContent, response.StatusCode)
	})
}

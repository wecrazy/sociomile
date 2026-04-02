package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/apperror"
	"gorm.io/gorm"
)

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestErrorMapsKnownErrorTypes(t *testing.T) {
	app := fiber.New()
	app.Get("/app", func(c fiber.Ctx) error {
		return Error(c, apperror.New(fiber.StatusBadRequest, "invalid_request", "request is invalid"))
	})
	app.Get("/not-found", func(c fiber.Ctx) error {
		return Error(c, gorm.ErrRecordNotFound)
	})
	app.Get("/internal", func(c fiber.Ctx) error {
		return Error(c, errors.New("boom"))
	})

	for _, testCase := range []struct {
		path    string
		status  int
		code    string
		message string
	}{
		{path: "/app", status: http.StatusBadRequest, code: "invalid_request", message: "request is invalid"},
		{path: "/not-found", status: http.StatusNotFound, code: "not_found", message: "resource not found"},
		{path: "/internal", status: http.StatusInternalServerError, code: "internal_error", message: "internal server error"},
	} {
		request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		response, err := app.Test(request)
		require.NoError(t, err)
		defer response.Body.Close()

		var body errorEnvelope
		require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
		require.Equal(t, testCase.status, response.StatusCode)
		require.Equal(t, testCase.code, body.Error.Code)
		require.Equal(t, testCase.message, body.Error.Message)
	}
}

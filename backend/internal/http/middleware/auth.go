package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/wecrazy/sociomile/backend/internal/apperror"
	"github.com/wecrazy/sociomile/backend/internal/auth"
)

type contextKey string

const (
	claimsKey    contextKey = "claims"
	requestIDKey contextKey = "request_id"
)

// RequestID attaches a request identifier to each Fiber request.
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		requestID := c.Get("X-Request-Id")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Locals(requestIDKey, requestID)
		c.Set("X-Request-Id", requestID)
		return c.Next()
	}
}

// CORS adds permissive CORS headers for the local assessment environment.
func CORS() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id")
		c.Set("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}

// Auth validates a bearer token and stores its claims in the request context.
func Auth(secret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" {
			return apperror.New(fiber.StatusUnauthorized, "missing_token", "authorization header is required")
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return apperror.New(fiber.StatusUnauthorized, "invalid_token", "authorization header must use Bearer token")
		}

		claims, err := auth.ParseToken(secret, parts[1])
		if err != nil {
			return apperror.New(fiber.StatusUnauthorized, "invalid_token", "access token is invalid")
		}

		c.Locals(claimsKey, claims)
		return c.Next()
	}
}

// RequireRoles allows only users with one of the given roles to continue.
func RequireRoles(roles ...string) fiber.Handler {
	allowed := map[string]struct{}{}
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c fiber.Ctx) error {
		claims, ok := CurrentClaims(c)
		if !ok {
			return apperror.New(fiber.StatusUnauthorized, "missing_claims", "authenticated user context is missing")
		}

		if _, exists := allowed[claims.Role]; !exists {
			return apperror.New(fiber.StatusForbidden, "forbidden", "user does not have the required role")
		}

		return c.Next()
	}
}

// CurrentClaims returns the JWT claims stored for the current request.
func CurrentClaims(c fiber.Ctx) (*auth.Claims, bool) {
	claims, ok := c.Locals(claimsKey).(*auth.Claims)
	return claims, ok
}

// CurrentRequestID returns the request identifier stored for the current request.
func CurrentRequestID(c fiber.Ctx) string {
	requestID, _ := c.Locals(requestIDKey).(string)
	return requestID
}

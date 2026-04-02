package response

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/wecrazy/sociomile/backend/internal/apperror"
	"gorm.io/gorm"
)

// JSON wraps a successful payload in the standard response envelope.
func JSON(c fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(fiber.Map{"data": data})
}

// Paginated wraps list data together with pagination metadata.
func Paginated(c fiber.Ctx, data any, total int64, offset int, limit int, filters map[string]any) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": data,
		"meta": fiber.Map{
			"total":   total,
			"offset":  offset,
			"limit":   limit,
			"filters": filters,
		},
	})
}

// Error converts application and persistence errors into HTTP responses.
func Error(c fiber.Ctx, err error) error {
	var appError *apperror.AppError
	if errors.As(err, &appError) {
		return c.Status(appError.Status).JSON(fiber.Map{"error": appError})
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		notFound := apperror.New(fiber.StatusNotFound, "not_found", "resource not found")
		return c.Status(notFound.Status).JSON(fiber.Map{"error": notFound})
	}

	internal := apperror.New(fiber.StatusInternalServerError, "internal_error", "internal server error")
	return c.Status(internal.Status).JSON(fiber.Map{"error": internal})
}

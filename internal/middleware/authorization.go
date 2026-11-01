package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

func RequireRole(repo *AccessRepository, roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, ok := UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "authentication_required",
			})
		}

		ctx, cancel := context.WithTimeout(
			context.Background(),
			3*time.Second,
		)
		defer cancel()

		allowed, err := repo.HasAnyRole(ctx, userID, roles)
		if err != nil {
			slog.Error("failed to authorize user", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal_server_error",
			})
		}

		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient_permissions",
			})
		}

		return c.Next()
	}
}

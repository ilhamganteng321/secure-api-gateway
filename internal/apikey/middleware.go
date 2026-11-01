package apikey

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

var ErrInvalidAPIKey = errors.New("invalid api key")

func Middleware(repo Repository) fiber.Handler {
	return func(c fiber.Ctx) error {
		rawKey := c.Get("X-API-Key")

		if rawKey == "" {
			return fiber.NewError(
				fiber.StatusUnauthorized,
				"missing api key",
			)
		}

		prefix, secret, found := strings.Cut(rawKey, ".")

		if !found || secret == "" {
			return fiber.NewError(
				fiber.StatusUnauthorized,
				"invalid api key",
			)
		}

		parts := strings.Split(prefix, "_")

		if len(parts) != 3 ||
			parts[0] != "sk" ||
			parts[1] != "live" ||
			len(parts[2]) != 16 {
			return fiber.NewError(
				fiber.StatusUnauthorized,
				"invalid api key",
			)
		}

		keyID := parts[2]

		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
		defer cancel()

		record, err := repo.FindByKeyID(ctx, keyID)
		if err != nil || record == nil || !record.Active {
			return fiber.NewError(
				fiber.StatusUnauthorized,
				"invalid api key",
			)
		}

		providedHash := sha256.Sum256([]byte(secret))

		if len(record.KeyHash) != sha256.Size ||
			subtle.ConstantTimeCompare(
				providedHash[:],
				record.KeyHash,
			) != 1 {
			return fiber.NewError(
				fiber.StatusUnauthorized,
				"invalid api key",
			)
		}

		c.Locals("api_client_id", record.ClientID)
		c.Locals("api_scopes", record.Scopes)
		c.Locals("api_rate_limit", record.RateLimit)
		c.Locals("api_rate_window", record.RateWindow)

		return c.Next()
	}
}

func RequireScope(requiredScope string) fiber.Handler {
	return func(c fiber.Ctx) error {
		scopes, ok := c.Locals("api_scopes").([]string)

		if !ok {
			return fiber.NewError(
				fiber.StatusForbidden,
				"insufficient permissions",
			)
		}

		for _, scope := range scopes {
			if scope == requiredScope || scope == "*" {
				return c.Next()
			}
		}

		return fiber.NewError(
			fiber.StatusForbidden,
			"insufficient permissions",
		)
	}
}

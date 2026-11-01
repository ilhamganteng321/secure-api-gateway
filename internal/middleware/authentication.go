package middleware

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/ilhamganteng321/secure-api-gateway/internal/modules/audit"
)

type TokenVerifier interface {
	VerifyAccessToken(token string) (int64, error)
}

type Authentication struct {
	tokens TokenVerifier
	repo   *AccessRepository
	audit  audit.Recorder
}

func NewAuthentication(
	tokens TokenVerifier,
	repo *AccessRepository,
	recorder audit.Recorder,
) *Authentication {
	return &Authentication{
		tokens: tokens,
		repo:   repo,
		audit:  recorder,
	}
}

func (a *Authentication) Required() fiber.Handler {
	return func(c fiber.Ctx) error {
		parts := strings.SplitN(
			c.Get("Authorization"),
			" ",
			2,
		)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") ||
			strings.TrimSpace(parts[1]) == "" {
			audit.RecordRequest(
				c,
				a.audit,
				"auth.access_denied",
				"denied",
				fiber.StatusUnauthorized,
				nil,
				map[string]any{"reason": "missing_or_invalid_authorization_header"},
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing_or_invalid_authorization_header",
			})
		}

		userID, err := a.tokens.VerifyAccessToken(strings.TrimSpace(parts[1]))
		if err != nil || userID <= 0 {
			audit.RecordRequest(
				c,
				a.audit,
				"auth.access_denied",
				"denied",
				fiber.StatusUnauthorized,
				nil,
				map[string]any{"reason": "invalid_or_expired_token"},
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid_or_expired_access_token",
			})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		active, err := a.repo.IsUserActive(ctx, userID)
		if err != nil {
			slog.Error("failed to check user status", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal_server_error",
			})
		}

		if !active {
			audit.RecordRequest(
				c,
				a.audit,
				"auth.access_denied",
				"denied",
				fiber.StatusUnauthorized,
				&userID,
				map[string]any{"reason": "inactive_user"},
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "user_inactive",
			})
		}

		c.Locals("userID", userID)
		return c.Next()
	}
}

func UserID(c fiber.Ctx) (int64, bool) {
	userID, ok := c.Locals("userID").(int64)
	return userID, ok && userID > 0
}

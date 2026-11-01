package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

func CORS(allowedOrigins []string) fiber.Handler {
	allowlist := make(map[string]struct{}, len(allowedOrigins))

	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowlist[origin] = struct{}{}
		}
	}

	return func(c fiber.Ctx) error {
		origin := c.Get("Origin")

		_, allowed := allowlist[origin]

		if origin != "" && allowed {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Vary", "Origin")
			c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Set("Access-Control-Max-Age", "600")
		}

		if c.Method() == fiber.MethodOptions {
			if !allowed {
				return c.SendStatus(fiber.StatusForbidden)
			}

			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}

package gateway

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

type RequestPolicy struct {
	MaxBodySize    int
	AllowedMethods map[string]bool
}

func NewRequestPolicy() *RequestPolicy {
	return &RequestPolicy{
		MaxBodySize: 1 * 1024 * 1024,
		AllowedMethods: map[string]bool{
			"GET":     true,
			"POST":    true,
			"PUT":     true,
			"PATCH":   true,
			"DELETE":  true,
			"HEAD":    true,
			"OPTIONS": true,
		},
	}
}

func (p *RequestPolicy) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if !p.AllowedMethods[strings.ToUpper(c.Method())] {
			return c.Status(fiber.StatusMethodNotAllowed).JSON(
				fiber.Map{
					"error": "method_not_allowed",
				},
			)
		}

		if len(c.Body()) > p.MaxBodySize {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(
				fiber.Map{
					"error": "request_body_too_large",
				},
			)
		}

		return c.Next()
	}
}

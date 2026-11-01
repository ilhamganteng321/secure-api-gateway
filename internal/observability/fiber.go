package observability

import (
	"github.com/gofiber/contrib/v3/otel"
	"github.com/gofiber/fiber/v3"
)

func TracingMiddleware() fiber.Handler {
	return otel.New()
}

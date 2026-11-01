package observability

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

func GatewayMetricsMiddleware(
	metrics *GatewayMetrics,
) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		if metrics != nil {
			metrics.ActiveRequests.Inc()
		}

		defer func() {
			if metrics != nil {
				metrics.ActiveRequests.Dec()
			}
		}()

		err := c.Next()

		status := c.Response().StatusCode()

		if err != nil {
			status = fiber.StatusInternalServerError
		}

		route := c.Route().Path
		if route == "" {
			route = "unmatched"
		}

		if metrics != nil {
			metrics.RecordRequest(
				c.Method(),
				route,
				status,
				time.Since(start),
			)
		}

		return err
	}
}

func RecordRateLimitRejection(
	metrics *GatewayMetrics,
	clientID string,
) {
	if metrics == nil {
		return
	}

	metrics.RateLimitRejections.WithLabelValues(clientID).Inc()
}

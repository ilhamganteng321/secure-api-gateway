package observability

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

func generateRequestID() (string, error) {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		requestID, err := generateRequestID()
		if err != nil {
			return fiber.NewError(
				fiber.StatusInternalServerError,
				"failed to generate request id",
			)
		}

		c.Locals("request_id", requestID)
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}

func MetricsMiddleware(metrics *Metrics) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		metrics.RequestsInFlight.Inc()
		defer metrics.RequestsInFlight.Dec()

		err := c.Next()

		duration := time.Since(start).Seconds()

		route := c.Route().Path
		if route == "" {
			route = "unmatched"
		}

		method := c.Method()
		status := strconv.Itoa(c.Response().StatusCode())

		metrics.RequestsTotal.WithLabelValues(
			method,
			route,
			status,
		).Inc()

		metrics.RequestDuration.WithLabelValues(
			method,
			route,
		).Observe(duration)

		return err
	}
}

func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		requestID, _ := c.Locals("request_id").(string)

		logger.Info("http request",
			"request_id", requestID,
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.IP(),
		)

		return err
	}
}

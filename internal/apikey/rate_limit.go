package apikey

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/ilhamganteng321/secure-api-gateway/internal/observability"
)

var clientRateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])

if current == 1 then
    redis.call("EXPIRE", KEYS[1], ARGV[1])
end

local ttl = redis.call("TTL", KEYS[1])

return {current, ttl}
`)

func ClientRateLimit(client *redis.Client, metrics ...*observability.GatewayMetrics) fiber.Handler {
	var gatewayMetrics *observability.GatewayMetrics
	if len(metrics) > 0 {
		gatewayMetrics = metrics[0]
	}

	return func(c fiber.Ctx) error {
		clientID, ok := c.Locals("api_client_id").(int64)
		if !ok || clientID <= 0 {
			return fiber.NewError(
				fiber.StatusUnauthorized,
				"invalid api client",
			)
		}

		limit, ok := c.Locals("api_rate_limit").(int)
		if !ok || limit <= 0 {
			return fiber.NewError(
				fiber.StatusInternalServerError,
				"invalid rate limit configuration",
			)
		}

		window, ok := c.Locals("api_rate_window").(int)
		if !ok || window <= 0 {
			return fiber.NewError(
				fiber.StatusInternalServerError,
				"invalid rate window configuration",
			)
		}

		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
		defer cancel()

		key := fmt.Sprintf("gateway:client:%d:rate", clientID)

		result, err := clientRateLimitScript.Run(
			ctx,
			client,
			[]string{key},
			window,
		).Slice()

		if err != nil {
			return fiber.NewError(
				fiber.StatusServiceUnavailable,
				"rate limiter unavailable",
			)
		}

		if len(result) != 2 {
			return fiber.NewError(
				fiber.StatusServiceUnavailable,
				"invalid rate limiter response",
			)
		}

		current, err := redisInt64(result[0])
		if err != nil {
			return fiber.NewError(
				fiber.StatusServiceUnavailable,
				"invalid rate limiter response",
			)
		}

		ttl, err := redisInt64(result[1])
		if err != nil {
			return fiber.NewError(
				fiber.StatusServiceUnavailable,
				"invalid rate limiter response",
			)
		}

		remaining := int64(limit) - current
		if remaining < 0 {
			remaining = 0
		}

		c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		if current > int64(limit) {
			observability.RecordRateLimitRejection(
				gatewayMetrics,
				strconv.FormatInt(clientID, 10),
			)

			c.Set("Retry-After", strconv.FormatInt(ttl, 10))

			return fiber.NewError(
				fiber.StatusTooManyRequests,
				"client rate limit exceeded",
			)
		}

		return c.Next()
	}
}

func redisInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported redis integer type %T", value)
	}
}

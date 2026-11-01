package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/ilhamganteng321/secure-api-gateway/internal/observability"
)

type RateLimiter struct {
	client  redis.UniversalClient
	limit   int64
	window  time.Duration
	prefix  string
	metrics *observability.GatewayMetrics
}

func NewRateLimiter(
	client redis.UniversalClient,
	limit int64,
	window time.Duration,
	prefix string,
) *RateLimiter {
	return &RateLimiter{
		client: client,
		limit:  limit,
		window: window,
		prefix: prefix,
	}
}

func (r *RateLimiter) SetMetrics(metrics *observability.GatewayMetrics) {
	r.metrics = metrics
}

// INCR dan EXPIRE dilakukan sebagai satu operasi atomik.
// PTTL mengembalikan sisa waktu window dalam milidetik.
var rateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])

if count == 1 then
    redis.call("PEXPIRE", KEYS[1], ARGV[1])
end

local ttl = redis.call("PTTL", KEYS[1])
return {count, ttl}
`)

func (r *RateLimiter) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if r.client == nil || r.limit <= 0 || r.window <= 0 {
			slog.Error("rate limiter is not configured correctly")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal_server_error",
			})
		}

		// Gunakan IP yang disediakan Fiber. Pastikan trusted proxy
		// dikonfigurasi dengan benar jika aplikasi berada di belakang proxy.
		ip := strings.TrimSpace(c.IP())
		if ip == "" {
			ip = "unknown"
		}

		ipHash := sha256.Sum256([]byte(ip))
		key := fmt.Sprintf(
			"%s:%s",
			r.prefix,
			hex.EncodeToString(ipHash[:]),
		)

		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
		defer cancel()

		result, err := rateLimitScript.Run(
			ctx,
			r.client,
			[]string{key},
			r.window.Milliseconds(),
		).Result()

		if err != nil {
			// Fail closed: jika Redis gagal, request tidak diteruskan.
			slog.Error("rate limiter Redis error", "error", err)
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "rate_limiter_unavailable",
			})
		}

		values, ok := result.([]interface{})
		if !ok || len(values) != 2 {
			slog.Error("unexpected Redis rate limiter response")
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "rate_limiter_unavailable",
			})
		}

		count, errCount := strconv.ParseInt(fmt.Sprint(values[0]), 10, 64)
		ttlMs, errTTL := strconv.ParseInt(fmt.Sprint(values[1]), 10, 64)

		if errCount != nil || errTTL != nil {
			slog.Error("failed to parse Redis rate limiter response")
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "rate_limiter_unavailable",
			})
		}

		remaining := r.limit - count
		if remaining < 0 {
			remaining = 0
		}

		if ttlMs < 0 {
			ttlMs = r.window.Milliseconds()
		}

		resetSeconds := int64((time.Duration(ttlMs) * time.Millisecond).Seconds())
		if ttlMs > 0 && resetSeconds == 0 {
			resetSeconds = 1
		}

		c.Set("X-RateLimit-Limit", strconv.FormatInt(r.limit, 10))
		c.Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(resetSeconds, 10))

		if count > r.limit {
			retryAfter := resetSeconds
			if retryAfter < 1 {
				retryAfter = 1
			}

			observability.RecordRateLimitRejection(r.metrics, "anonymous")
			c.Set("Retry-After", strconv.FormatInt(retryAfter, 10))

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate_limit_exceeded",
				"retry_after": retryAfter,
			})
		}

		return c.Next()
	}
}

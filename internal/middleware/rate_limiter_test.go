package middleware_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/ilhamganteng321/secure-api-gateway/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func TestRateLimiter_RedisUnavailable_Returns503(t *testing.T) {
	// Redis client pointing to an unreachable port to simulate Redis down/failure
	deadClient := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:54321",
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	defer deadClient.Close()

	limiter := middleware.NewRateLimiter(
		deadClient,
		5,
		time.Minute,
		"test:rate:unavailable",
	)

	app := fiber.New()
	app.Get("/test", limiter.Middleware(), func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("unexpected app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 Service Unavailable, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var res map[string]string
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if res["error"] != "rate_limiter_unavailable" {
		t.Errorf("expected error 'rate_limiter_unavailable', got '%s'", res["error"])
	}
}

func TestRateLimiter_Misconfigured_Returns500(t *testing.T) {
	limiter := middleware.NewRateLimiter(nil, 0, 0, "")

	app := fiber.New()
	app.Get("/test", limiter.Middleware(), func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500 Internal Server Error, got %d", resp.StatusCode)
	}
}

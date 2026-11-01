package apikey_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/ilhamganteng321/secure-api-gateway/internal/apikey"
)

func TestClientRateLimit(t *testing.T) {
	addr := "localhost:6379"

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       15,
	})

	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}

	defer client.Close()

	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("api_client_id", int64(999999))
		c.Locals("api_rate_limit", 2)
		c.Locals("api_rate_window", 60)
		return c.Next()
	})

	app.Use(apikey.ClientRateLimit(client))

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	for i := 1; i <= 3; i++ {
		resp, err := app.Test(
			httptest.NewRequest("GET", "/test", nil),
		)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()

		if i <= 2 && resp.StatusCode != 200 {
			t.Fatalf("request %d: expected 200, got %d", i, resp.StatusCode)
		}

		if i == 3 && resp.StatusCode != 429 {
			t.Fatalf("request %d: expected 429, got %d", i, resp.StatusCode)
		}
	}
}

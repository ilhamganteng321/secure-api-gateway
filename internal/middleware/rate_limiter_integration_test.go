//go:build integration

package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/ilhamganteng321/secure-api-gateway/internal/middleware"
)

func setupRedis(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	db := 15
	if dbStr := os.Getenv("TEST_REDIS_DB"); dbStr != "" {
		if parsed, err := strconv.Atoi(dbStr); err == nil {
			db = parsed
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     os.Getenv("TEST_REDIS_PASSWORD"),
		DB:           db,
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})

	if err := client.Ping(t.Context()).Err(); err != nil {
		client.Close()
		t.Fatalf("Redis integration test requires Redis at %s: %v", addr, err)
	}

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

func TestRateLimiter_EnforcesLimit(t *testing.T) {
	client := setupRedis(t)

	prefix := fmt.Sprintf("test:rate:%d", time.Now().UnixNano())

	limiter := middleware.NewRateLimiter(
		client,
		2,
		time.Minute,
		prefix,
	)

	app := fiber.New()
	app.Get("/test", limiter.Middleware(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	for i, wantStatus := range []int{
		http.StatusOK,
		http.StatusOK,
		http.StatusTooManyRequests,
	} {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}

		resp.Body.Close()

		if resp.StatusCode != wantStatus {
			t.Fatalf(
				"request %d: got status %d, want %d",
				i+1,
				resp.StatusCode,
				wantStatus,
			)
		}

		if i == 2 && resp.Header.Get("Retry-After") == "" {
			t.Fatal("expected Retry-After header on HTTP 429")
		}
	}
}

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	client := setupRedis(t)

	prefix := fmt.Sprintf("test:rate:concurrent:%d", time.Now().UnixNano())

	limiter := middleware.NewRateLimiter(
		client,
		5,
		time.Minute,
		prefix,
	)

	app := fiber.New()
	app.Get("/test", limiter.Middleware(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	const totalRequests = 20

	results := make(chan int, totalRequests)

	for i := 0; i < totalRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			resp, err := app.Test(req)
			if err != nil {
				results <- 0
				return
			}

			defer resp.Body.Close()
			results <- resp.StatusCode
		}()
	}

	successCount := 0
	limitedCount := 0

	for i := 0; i < totalRequests; i++ {
		status := <-results

		switch status {
		case http.StatusOK:
			successCount++
		case http.StatusTooManyRequests:
			limitedCount++
		default:
			t.Errorf("unexpected HTTP status: %d", status)
		}
	}

	if successCount != 5 {
		t.Errorf("got %d successful requests, want 5", successCount)
	}

	if limitedCount != 15 {
		t.Errorf("got %d rate-limited requests, want 15", limitedCount)
	}
}

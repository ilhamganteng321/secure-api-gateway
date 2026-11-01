//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ilhamganteng321/secure-api-gateway/internal/app"
	"github.com/ilhamganteng321/secure-api-gateway/internal/config"
	"github.com/ilhamganteng321/secure-api-gateway/internal/observability"
)

func setupIntegrationApp(t *testing.T) *fiber.App {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Fatalf("ping test database: %v", err)
	}
	t.Cleanup(db.Close)

	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisDB := 15
	if dbStr := os.Getenv("TEST_REDIS_DB"); dbStr != "" {
		if parsed, err := strconv.Atoi(dbStr); err == nil {
			redisDB = parsed
		}
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("TEST_REDIS_PASSWORD"),
		DB:       redisDB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisClient.Close()
		db.Close()
		t.Skipf("Redis unavailable: %v", err)
	}
	t.Cleanup(func() { redisClient.Close() })

	cfg := config.Config{
		AppName:              "secure-api-gateway-test",
		AppEnv:               "test",
		AppPort:              "3000",
		DatabaseURL:          dsn,
		RedisAddr:            redisAddr,
		RedisPassword:        os.Getenv("TEST_REDIS_PASSWORD"),
		RedisDB:              15,
		JWTIssuer:            "secure-api-gateway",
		JWTAudience:          "secure-api-client",
		JWTSecret:            "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=",
		AccessTokenTTL:       15 * time.Minute,
		RefreshTokenTTL:      168 * time.Hour,
		OTELServiceName:      "secure-api-gateway-test",
		OTELExporterEndpoint: "localhost:4318",
		OTELTracesSamplerArg: 1.0,
	}

	return app.New(app.Dependencies{
		Config: cfg,
		DB:     db,
		Redis:  redisClient,
		Logger: observability.NewLogger("test"),
	})
}

func uniqueEmail(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("integration-%d@example.invalid", time.Now().UnixNano())
}

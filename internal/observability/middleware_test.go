package observability_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/ilhamganteng321/secure-api-gateway/internal/observability"
)

func TestRequestIDMiddleware(t *testing.T) {
	app := fiber.New()

	app.Use(observability.RequestID())

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(
		httptest.NewRequest("GET", "/test", nil),
	)

	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	requestID := resp.Header.Get("X-Request-ID")

	if requestID == "" {
		t.Fatal("expected X-Request-ID header")
	}

	if len(requestID) != 32 {
		t.Fatalf("expected 32 character request ID, got %d", len(requestID))
	}
}

func TestMetricsMiddleware(t *testing.T) {
	app := fiber.New()

	metrics := observability.NewMetrics()

	app.Use(observability.MetricsMiddleware(metrics))

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(
		httptest.NewRequest("GET", "/test", nil),
	)

	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	metricFamilies, err := metrics.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}

	found := false

	for _, family := range metricFamilies {
		if family.GetName() == "http_requests_total" {
			found = true

			if len(family.Metric) == 0 {
				t.Fatal("expected at least one HTTP request metric")
			}
		}
	}

	if !found {
		t.Fatal("http_requests_total metric not found")
	}
}

func TestRequestIDIsUnique(t *testing.T) {
	app := fiber.New()

	app.Use(observability.RequestID())

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	first, err := app.Test(httptest.NewRequest("GET", "/test", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer first.Body.Close()

	second, err := app.Test(httptest.NewRequest("GET", "/test", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer second.Body.Close()

	firstID := first.Header.Get("X-Request-ID")
	secondID := second.Header.Get("X-Request-ID")

	if firstID == "" || secondID == "" {
		t.Fatal("request ID must not be empty")
	}

	if strings.EqualFold(firstID, secondID) {
		t.Fatal("request IDs should be unique")
	}
}

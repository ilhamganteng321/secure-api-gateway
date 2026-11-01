package observability_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/ilhamganteng321/secure-api-gateway/internal/observability"
)

func TestMetricsEndpoint(t *testing.T) {
	app := fiber.New()
	metrics := observability.NewMetrics()

	app.Get("/metrics", metrics.Handler())

	app.Use(observability.MetricsMiddleware(metrics))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("metrics status = %d, want 200", resp.StatusCode)
	}

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "http_requests_total") {
		t.Fatalf("metrics body missing http_requests_total: %s", body)
	}
}

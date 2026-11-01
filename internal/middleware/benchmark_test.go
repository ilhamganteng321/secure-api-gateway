package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func BenchmarkHealthEndpoint(b *testing.B) {
	app := fiber.New()

	app.Get("/health/live", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := app.Test(newBenchmarkRequest())
		if err != nil {
			b.Fatal(err)
		}

		resp.Body.Close()

		if resp.StatusCode != 204 {
			b.Fatalf("status = %d, want 204", resp.StatusCode)
		}
	}
}

func newBenchmarkRequest() *http.Request {
	return httptest.NewRequest("GET", "/health/live", nil)
}

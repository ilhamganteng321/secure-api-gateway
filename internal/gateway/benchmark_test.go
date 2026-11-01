package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func BenchmarkGatewayHealthEndpoint(b *testing.B) {
	app := fiber.New()

	app.Get("/benchmark", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "secure-api-gateway",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/benchmark", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}

		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	}
}

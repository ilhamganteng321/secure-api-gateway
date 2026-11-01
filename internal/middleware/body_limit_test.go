package middleware_test

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestBodyLimit_RejectsOversizedPayload(t *testing.T) {
	app := fiber.New(fiber.Config{
		BodyLimit: 16,
	})
	app.Post("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	body := strings.Repeat("a", 64)
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "text/plain")
	resp, err := app.Test(req)
	if err != nil {
		if !strings.Contains(err.Error(), "body size exceeds") {
			t.Fatalf("app.Test() error = %v", err)
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusRequestEntityTooLarge)
	}
}

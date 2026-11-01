package apikey_test

import (
	"context"
	"crypto/sha256"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/ilhamganteng321/secure-api-gateway/internal/apikey"
)

type fakeRepository struct {
	record *apikey.KeyRecord
}

func (f *fakeRepository) FindByKeyID(
	ctx context.Context,
	keyID string,
) (*apikey.KeyRecord, error) {
	return f.record, nil
}

func TestGenerateAPIKey(t *testing.T) {
	generated, err := apikey.Generate()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(generated.Plaintext, "sk_live_") {
		t.Fatal("invalid API key prefix")
	}

	if len(generated.KeyID) != 16 {
		t.Fatal("invalid key ID length")
	}

	if len(generated.Hash) != sha256.Size {
		t.Fatal("invalid hash length")
	}

	if generated.Plaintext == "" {
		t.Fatal("plaintext API key must not be empty")
	}
}

func TestAPIKeyAuthentication(t *testing.T) {
	generated, err := apikey.Generate()
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakeRepository{
		record: &apikey.KeyRecord{
			ClientID: 10,
			KeyHash:  generated.Hash,
			Scopes:   []string{"products:read"},
			Active:   true,
		},
	}

	app := fiber.New()

	app.Get(
		"/protected",
		apikey.Middleware(repo),
		apikey.RequireScope("products:read"),
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"client_id": c.Locals("api_client_id"),
			})
		},
	)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-API-Key", generated.Plaintext)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIKeyRejectsInvalidKey(t *testing.T) {
	repo := &fakeRepository{
		record: nil,
	}

	app := fiber.New()

	app.Get(
		"/protected",
		apikey.Middleware(repo),
		func(c fiber.Ctx) error {
			return c.SendStatus(200)
		},
	)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set(
		"X-API-Key",
		"sk_live_invalid.invalid",
	)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAPIKeyRejectsInsufficientScope(t *testing.T) {
	generated, err := apikey.Generate()
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakeRepository{
		record: &apikey.KeyRecord{
			ClientID: 10,
			KeyHash:  generated.Hash,
			Scopes:   []string{"products:read"},
			Active:   true,
		},
	}

	app := fiber.New()

	app.Get(
		"/protected",
		apikey.Middleware(repo),
		apikey.RequireScope("orders:write"),
		func(c fiber.Ctx) error {
			return c.SendStatus(200)
		},
	)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-API-Key", generated.Plaintext)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

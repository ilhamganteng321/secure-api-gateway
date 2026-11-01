//go:build integration

package middleware_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilhamganteng321/secure-api-gateway/internal/middleware"
)

type fakeTokenVerifier struct {
	tokens map[string]int64
}

func (f fakeTokenVerifier) VerifyAccessToken(token string) (int64, error) {
	userID, ok := f.tokens[token]
	if !ok {
		return 0, errors.New("invalid token")
	}
	return userID, nil
}

func setupAccessDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}

	if err := db.Ping(t.Context()); err != nil {
		db.Close()
		t.Fatalf("ping test database: %v", err)
	}

	t.Cleanup(db.Close)
	return db
}

func createAccessTestUser(t *testing.T, db *pgxpool.Pool, role string, active bool) int64 {
	t.Helper()

	ctx := t.Context()
	email := fmt.Sprintf(
		"access-test-%d@example.invalid",
		time.Now().UnixNano(),
	)

	var userID int64
	err := db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, is_active)
		VALUES ($1, 'integration-test-hash', $2)
		RETURNING id
	`, email, active).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE name = $2
	`, userID, role)
	if err != nil {
		t.Fatalf("assign role %q: %v", role, err)
	}

	t.Cleanup(func() {
		cleanupCtx := t.Context()
		_, _ = db.Exec(cleanupCtx, `DELETE FROM audit_events WHERE actor_user_id = $1`, userID)
		_, _ = db.Exec(cleanupCtx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
		_, _ = db.Exec(cleanupCtx, `DELETE FROM user_roles WHERE user_id = $1`, userID)
		_, _ = db.Exec(cleanupCtx, `DELETE FROM users WHERE id = $1`, userID)
	})

	return userID
}

func TestAccessControl_RoleAndAuthentication(t *testing.T) {
	db := setupAccessDB(t)
	repo := middleware.NewAccessRepository(db)

	userID := createAccessTestUser(t, db, "user", true)
	adminID := createAccessTestUser(t, db, "admin", true)
	inactiveID := createAccessTestUser(t, db, "user", false)

	verifier := fakeTokenVerifier{
		tokens: map[string]int64{
			"user-token":     userID,
			"admin-token":    adminID,
			"inactive-token": inactiveID,
		},
	}

	authn := middleware.NewAuthentication(verifier, repo, nil)

	app := fiber.New()
	app.Get("/me", authn.Required(), func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	app.Get(
		"/admin",
		authn.Required(),
		middleware.RequireRole(repo, "admin"),
		func(c fiber.Ctx) error {
			return c.SendStatus(http.StatusOK)
		},
	)

	tests := []struct {
		name       string
		path       string
		token      string
		wantStatus int
	}{
		{"missing token", "/me", "", http.StatusUnauthorized},
		{"invalid token", "/me", "invalid-token", http.StatusUnauthorized},
		{"valid user token", "/me", "user-token", http.StatusOK},
		{"user denied admin", "/admin", "user-token", http.StatusForbidden},
		{"admin allowed", "/admin", "admin-token", http.StatusOK},
		{"inactive user denied", "/me", "inactive-token", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("got HTTP %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

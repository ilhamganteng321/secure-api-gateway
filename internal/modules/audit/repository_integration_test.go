//go:build integration

package audit_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilhamganteng321/secure-api-gateway/internal/modules/audit"
)

func TestAuditRepository_RecordEvent(t *testing.T) {
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
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	repo := audit.NewRepository(db)
	requestID := fmt.Sprintf("audit-test-%d", time.Now().UnixNano())

	event := audit.Event{
		Type:       "auth.login",
		Outcome:    "failure",
		RequestID:  requestID,
		Method:     "POST",
		Path:       "/api/v1/auth/login",
		StatusCode: 401,
		ClientIP:   "127.0.0.1",
		Metadata: map[string]any{
			"reason": "invalid_credentials",
		},
	}

	if err := repo.Record(ctx, event); err != nil {
		t.Fatalf("record audit event: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			3*time.Second,
		)
		defer cleanupCancel()

		_, _ = db.Exec(
			cleanupCtx,
			`DELETE FROM audit_events WHERE request_id = $1`,
			requestID,
		)
	})

	var eventType string
	var outcome string
	var statusCode int
	var metadata map[string]any

	err = db.QueryRow(ctx, `
		SELECT event_type, outcome, status_code, metadata
		FROM audit_events
		WHERE request_id = $1
	`, requestID).Scan(
		&eventType,
		&outcome,
		&statusCode,
		&metadata,
	)
	if err != nil {
		t.Fatalf("query audit event: %v", err)
	}

	if eventType != "auth.login" {
		t.Errorf("got event type %q, want auth.login", eventType)
	}
	if outcome != "failure" {
		t.Errorf("got outcome %q, want failure", outcome)
	}
	if statusCode != 401 {
		t.Errorf("got status code %d, want 401", statusCode)
	}
	if metadata["reason"] != "invalid_credentials" {
		t.Errorf("unexpected audit metadata: %#v", metadata)
	}
}

func TestAuditRepository_RejectsInvalidOutcome(t *testing.T) {
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
	defer db.Close()

	repo := audit.NewRepository(db)

	err = repo.Record(ctx, audit.Event{
		Type:    "auth.login",
		Outcome: "maybe",
	})

	if err == nil {
		t.Fatal("expected invalid outcome to be rejected")
	}
}

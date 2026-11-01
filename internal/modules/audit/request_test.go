package audit_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/ilhamganteng321/secure-api-gateway/internal/modules/audit"
)

type mockFailingRecorder struct{}

func (m *mockFailingRecorder) Record(ctx context.Context, event audit.Event) error {
	return errors.New("simulated database failure")
}

func TestRecordRequest_FailureReportedViaSlog(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	originalDefault := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(originalDefault)

	app := fiber.New()
	app.Get("/test-audit-fail", func(c fiber.Ctx) error {
		audit.RecordRequest(
			c,
			&mockFailingRecorder{},
			"auth.test_failure",
			"denied",
			http.StatusUnauthorized,
			nil,
			map[string]any{"reason": "testing_audit_failure"},
		)
		return c.SendString("proceeded without crash")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-audit-fail", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "failed to persist audit event") {
		t.Errorf("expected log to contain 'failed to persist audit event', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "simulated database failure") {
		t.Errorf("expected log to contain 'simulated database failure', got: %s", logOutput)
	}
}

func TestRecordRequest_NilRecorderHandledGracefully(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	originalDefault := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(originalDefault)

	app := fiber.New()
	app.Get("/test-nil-audit", func(c fiber.Ctx) error {
		audit.RecordRequest(
			c,
			nil,
			"auth.test_nil",
			"denied",
			http.StatusUnauthorized,
			nil,
			nil,
		)
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-nil-audit", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "audit recorder is not configured") {
		t.Errorf("expected log to contain 'audit recorder is not configured', got: %s", logOutput)
	}
}

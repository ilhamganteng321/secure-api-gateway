package health

import (
	"context"
	"testing"
)

func TestCheckerNilDependencies(t *testing.T) {
	checker := &Checker{}

	result := checker.Check(context.Background())

	if result.Status != "degraded" {
		t.Fatalf("expected degraded, got %s", result.Status)
	}

	if result.Dependencies["postgres"].Status != "error" {
		t.Fatal("expected postgres error")
	}

	if result.Dependencies["redis"].Status != "error" {
		t.Fatal("expected redis error")
	}
}

func TestCheckerNoDependenciesConfigured(t *testing.T) {
	checker := &Checker{}

	result := checker.Check(context.Background())

	if result.Status == "ok" {
		t.Fatal("expected non-ok status")
	}
}

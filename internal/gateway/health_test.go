package gateway_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ilhamganteng321/secure-api-gateway/internal/gateway"
)

func TestHealthChecker(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()

	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer unhealthy.Close()

	checker := gateway.NewHealthChecker(100 * time.Millisecond)

	if !checker.Check(context.Background(), "healthy", healthy.URL+"/health") {
		t.Fatal("expected healthy upstream")
	}

	if checker.Check(context.Background(), "unhealthy", unhealthy.URL+"/health") {
		t.Fatal("expected unhealthy upstream")
	}
}

func TestHealthCheckerContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := gateway.NewHealthChecker(1 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if checker.Check(ctx, "slow", server.URL+"/health") {
		t.Fatal("expected cancelled context to fail health check")
	}
}

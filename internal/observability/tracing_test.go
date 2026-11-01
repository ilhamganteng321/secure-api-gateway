package observability

import (
	"context"
	"testing"
)

func TestInitTracingShutdown(t *testing.T) {
	shutdown, err := InitTracing(context.Background(), TracingConfig{
		ServiceName:   "secure-api-gateway-test",
		Endpoint:      "localhost:4318",
		SamplingRatio: 1.0,
	})
	if err != nil {
		t.Fatalf("InitTracing returned error: %v", err)
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("tracing shutdown returned error: %v", err)
	}
}

func TestTracingMiddleware(t *testing.T) {
	handler := TracingMiddleware()
	if handler == nil {
		t.Fatal("TracingMiddleware returned nil")
	}
}

package gateway_test

import (
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"

	"github.com/ilhamganteng321/secure-api-gateway/internal/gateway"
)

func TestUpstreamBreakerOpens(t *testing.T) {
	breaker := gateway.NewUpstreamBreaker("users")

	fail := errors.New("upstream failed")

	for i := 0; i < 5; i++ {
		_, _ = breaker.Execute(func() ([]byte, error) {
			return nil, fail
		})
	}

	if breaker.State() != gobreaker.StateOpen {
		t.Fatalf("expected circuit to be open, got %v", breaker.State())
	}

	_, err := breaker.Execute(func() ([]byte, error) {
		return []byte("ok"), nil
	})
	if err != gateway.ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestUpstreamBreakerHalfOpen(t *testing.T) {
	breaker := gateway.NewUpstreamBreaker("slow-users")

	fail := errors.New("upstream failed")

	for i := 0; i < 5; i++ {
		_, _ = breaker.Execute(func() ([]byte, error) {
			return nil, fail
		})
	}

	time.Sleep(10100 * time.Millisecond)

	result, err := breaker.Execute(func() ([]byte, error) {
		return []byte("recovered"), nil
	})
	if err != nil {
		t.Fatalf("expected half-open request to succeed, got %v", err)
	}
	if string(result) != "recovered" {
		t.Fatalf("unexpected result: %s", string(result))
	}
}

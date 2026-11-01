package resilience_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ilhamganteng321/secure-api-gateway/internal/resilience"
)

func TestDoWithTimeout(t *testing.T) {
	_, err := resilience.DoWithTimeout(
		context.Background(),
		20*time.Millisecond,
		func(ctx context.Context) (string, error) {
			<-ctx.Done()
			return "", ctx.Err()
		},
	)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestRetryEventuallySucceeds(t *testing.T) {
	attempts := 0

	err := resilience.Retry(
		context.Background(),
		resilience.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: time.Millisecond,
			MaxDelay:     5 * time.Millisecond,
		},
		func() error {
			attempts++

			if attempts < 3 {
				return errors.New("temporary failure")
			}

			return nil
		},
		func(err error) bool {
			return true
		},
	)

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryDoesNotRetryPermanentError(t *testing.T) {
	attempts := 0
	permanentErr := errors.New("invalid request")

	err := resilience.Retry(
		context.Background(),
		resilience.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: time.Millisecond,
			MaxDelay:     5 * time.Millisecond,
		},
		func() error {
			attempts++
			return permanentErr
		},
		func(err error) bool {
			return false
		},
	)

	if !errors.Is(err, permanentErr) {
		t.Fatalf("expected permanent error, got %v", err)
	}

	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	breaker := resilience.NewBreaker(
		"test-service",
		2,
		5*time.Second,
	)

	failure := errors.New("service unavailable")

	for i := 0; i < 2; i++ {
		err := breaker.Execute(func() error {
			return failure
		})

		if !errors.Is(err, failure) {
			t.Fatalf("expected service failure, got %v", err)
		}
	}

	err := breaker.Execute(func() error {
		t.Fatal("operation should not execute while breaker is open")
		return nil
	})

	if !errors.Is(err, resilience.ErrCircuitOpen) {
		t.Fatalf("expected circuit open, got %v", err)
	}
}

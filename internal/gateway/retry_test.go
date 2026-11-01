package gateway_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/ilhamganteng321/secure-api-gateway/internal/gateway"
)

func TestSafeRetryMethod(t *testing.T) {
	if !gateway.IsSafeRetryMethod("GET") {
		t.Fatal("GET should be retryable")
	}
	if gateway.IsSafeRetryMethod("POST") {
		t.Fatal("POST should not be retryable by default")
	}
	if gateway.IsSafeRetryMethod("DELETE") {
		t.Fatal("DELETE should not be automatically retryable")
	}
}

func TestTransientGatewayError(t *testing.T) {
	if !gateway.IsTransientGatewayError(context.DeadlineExceeded) {
		t.Fatal("deadline exceeded should be transient")
	}

	timeoutErr := &net.OpError{
		Op:  "dial",
		Err: &timeoutError{},
	}
	if !gateway.IsTransientGatewayError(timeoutErr) {
		t.Fatal("timeout should be transient")
	}

	if gateway.IsTransientGatewayError(errors.New("permanent")) {
		t.Fatal("permanent error should not be transient")
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

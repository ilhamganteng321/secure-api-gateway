package gateway

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
)

// IsSafeRetryMethod reports whether an HTTP method is safe to retry
// automatically. POST is intentionally excluded by default.
func IsSafeRetryMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// IsTransientGatewayError reports whether an error is likely temporary.
func IsTransientGatewayError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		return errors.Is(err, context.DeadlineExceeded)
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}

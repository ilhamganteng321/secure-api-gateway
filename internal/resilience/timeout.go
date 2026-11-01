package resilience

import (
	"context"
	"time"
)

func DoWithTimeout[T any](
	ctx context.Context,
	timeout time.Duration,
	fn func(context.Context) (T, error),
) (T, error) {
	var zero T

	if timeout <= 0 {
		return zero, context.DeadlineExceeded
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return fn(ctx)
}

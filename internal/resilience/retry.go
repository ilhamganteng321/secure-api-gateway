package resilience

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

func Retry(
	ctx context.Context,
	config RetryConfig,
	operation func() error,
	retryable func(error) bool,
) error {
	if config.MaxAttempts < 1 {
		return errors.New("max attempts must be at least 1")
	}

	if config.InitialDelay <= 0 || config.MaxDelay <= 0 {
		return errors.New("retry delays must be positive")
	}

	if config.InitialDelay > config.MaxDelay {
		return errors.New("initial delay cannot exceed max delay")
	}

	if operation == nil {
		return errors.New("operation cannot be nil")
	}

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := operation()
		if err == nil {
			return nil
		}

		if retryable == nil || !retryable(err) {
			return err
		}

		if attempt == config.MaxAttempts {
			return err
		}

		delay := config.InitialDelay

		for i := 1; i < attempt; i++ {
			if delay >= config.MaxDelay/2 {
				delay = config.MaxDelay
				break
			}
			delay *= 2
		}

		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		// Jitter: 50% sampai 150% dari delay.
		jitter := 0.5 + rand.Float64()
		wait := time.Duration(float64(delay) * jitter)

		timer := time.NewTimer(wait)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return nil
}

package resilience

import (
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type Breaker struct {
	cb *gobreaker.CircuitBreaker[struct{}]
}

func NewBreaker(
	name string,
	failureThreshold uint32,
	openDuration time.Duration,
) *Breaker {
	if failureThreshold == 0 {
		failureThreshold = 5
	}

	if openDuration <= 0 {
		openDuration = 10 * time.Second
	}

	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Timeout:     openDuration,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= failureThreshold
		},
	}

	return &Breaker{
		cb: gobreaker.NewCircuitBreaker[struct{}](settings),
	}
}

func (b *Breaker) Execute(operation func() error) error {
	if operation == nil {
		return errors.New("operation cannot be nil")
	}

	_, err := b.cb.Execute(func() (struct{}, error) {
		return struct{}{}, operation()
	})

	if errors.Is(err, gobreaker.ErrOpenState) ||
		errors.Is(err, gobreaker.ErrTooManyRequests) {
		return ErrCircuitOpen
	}

	return err
}

func (b *Breaker) State() gobreaker.State {
	return b.cb.State()
}

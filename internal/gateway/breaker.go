package gateway

import (
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"
)

var ErrCircuitOpen = errors.New("upstream circuit breaker is open")

type UpstreamBreaker struct {
	breaker *gobreaker.CircuitBreaker[[]byte]
}

func NewUpstreamBreaker(name string) *UpstreamBreaker {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.Requests >= 5 &&
				counts.TotalFailures >= 3
		},
	}

	return &UpstreamBreaker{
		breaker: gobreaker.NewCircuitBreaker[[]byte](settings),
	}
}

func (b *UpstreamBreaker) Execute(
	fn func() ([]byte, error),
) ([]byte, error) {
	result, err := b.breaker.Execute(fn)
	if errors.Is(err, gobreaker.ErrOpenState) ||
		errors.Is(err, gobreaker.ErrTooManyRequests) {
		return nil, ErrCircuitOpen
	}

	return result, err
}

func (b *UpstreamBreaker) State() gobreaker.State {
	return b.breaker.State()
}

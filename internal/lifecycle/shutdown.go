package lifecycle

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

type ShutdownFunc func(context.Context) error

type Manager struct {
	mu       sync.Mutex
	handlers []ShutdownFunc
	timeout  time.Duration
	logger   *slog.Logger
}

func NewManager(
	timeout time.Duration,
	logger *slog.Logger,
) *Manager {
	return &Manager{
		timeout: timeout,
		logger:  logger,
	}
}

func (m *Manager) Add(fn ShutdownFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.handlers = append(m.handlers, fn)
}

func (m *Manager) Shutdown(
	ctx context.Context,
) error {
	m.mu.Lock()
	handlers := append(
		[]ShutdownFunc(nil),
		m.handlers...,
	)
	m.mu.Unlock()

	var errs []error

	for i := len(handlers) - 1; i >= 0; i-- {
		shutdownCtx, cancel := context.WithTimeout(
			ctx,
			m.timeout,
		)

		if err := handlers[i](shutdownCtx); err != nil {
			errs = append(errs, err)

			if m.logger != nil {
				m.logger.Error(
					"shutdown failed",
					"error", err,
				)
			}
		}

		cancel()
	}

	return errors.Join(errs...)
}

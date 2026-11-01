package lifecycle

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestShutdownReverseOrder(t *testing.T) {
	manager := NewManager(1*time.Second, nil)

	var order []int

	for i := 1; i <= 3; i++ {
		n := i
		manager.Add(func(ctx context.Context) error {
			order = append(order, n)
			return nil
		})
	}

	if err := manager.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(order) != 3 || order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Fatalf("unexpected shutdown order: %v", order)
	}
}

func TestShutdownCollectsErrors(t *testing.T) {
	manager := NewManager(1*time.Second, nil)

	manager.Add(func(ctx context.Context) error {
		return errors.New("first error")
	})

	manager.Add(func(ctx context.Context) error {
		return errors.New("second error")
	})

	err := manager.Shutdown(context.Background())
	if err == nil {
		t.Fatal("expected shutdown error")
	}
}

func TestShutdownConcurrentRegistration(t *testing.T) {
	manager := NewManager(1*time.Second, nil)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			manager.Add(func(ctx context.Context) error {
				return nil
			})
		}()
	}

	wg.Wait()

	if err := manager.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

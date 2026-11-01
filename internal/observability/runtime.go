package observability

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

func StartRuntimeMonitor(ctx context.Context, logger *slog.Logger, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var mem runtime.MemStats
				runtime.ReadMemStats(&mem)

				logger.Info(
					"runtime stats",
					"goroutines", runtime.NumGoroutine(),
					"heap_alloc_mb", mem.HeapAlloc/1024/1024,
					"heap_sys_mb", mem.HeapSys/1024/1024,
					"gc_cycles", mem.NumGC,
				)
			}
		}
	}()
}

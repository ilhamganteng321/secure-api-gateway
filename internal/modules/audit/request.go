package audit

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Recorder interface {
	Record(ctx context.Context, event Event) error
}

func RecordRequest(
	c fiber.Ctx,
	recorder Recorder,
	eventType string,
	outcome string,
	statusCode int,
	actorUserID *int64,
	metadata map[string]any,
) {
	if recorder == nil {
		slog.Error("audit recorder is not configured")
		return
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	event := Event{
		ActorUserID: actorUserID,
		Type:        eventType,
		Outcome:     outcome,
		RequestID:   c.Get("X-Request-ID"),
		Method:      c.Method(),
		Path:        c.Path(),
		StatusCode:  statusCode,
		ClientIP:    c.IP(),
		Metadata:    metadata,
	}

	if err := recorder.Record(ctx, event); err != nil {
		// Kegagalan audit tidak boleh menimpa respons asli.
		slog.Error(
			"failed to persist audit event",
			"event_type", eventType,
			"error", err,
		)
	}
}

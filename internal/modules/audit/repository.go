package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ActorUserID *int64
	Type        string
	Outcome     string
	RequestID   string
	Method      string
	Path        string
	StatusCode  int
	ClientIP    string
	Metadata    map[string]any
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Record(ctx context.Context, event Event) error {
	if event.Type == "" {
		return fmt.Errorf("audit event type is required")
	}

	switch event.Outcome {
	case "success", "failure", "denied":
	default:
		return fmt.Errorf("invalid audit outcome")
	}

	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	var actor any
	if event.ActorUserID != nil && *event.ActorUserID > 0 {
		actor = *event.ActorUserID
	}

	var clientIP any
	if ip := net.ParseIP(strings.TrimSpace(event.ClientIP)); ip != nil {
		clientIP = ip.String()
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO audit_events (
			actor_user_id,
			event_type,
			outcome,
			request_id,
			http_method,
			request_path,
			status_code,
			client_ip,
			metadata
		)
		VALUES (
			$1, $2, $3, NULLIF($4, ''),
			NULLIF($5, ''), NULLIF($6, ''),
			NULLIF($7, 0), $8, $9::jsonb
		)
	`,
		actor,
		event.Type,
		event.Outcome,
		event.RequestID,
		event.Method,
		event.Path,
		event.StatusCode,
		clientIP,
		string(encoded),
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}

	return nil
}

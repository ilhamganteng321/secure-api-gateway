package apikey

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type KeyRecord struct {
	ClientID   int64
	KeyHash    []byte
	Scopes     []string
	Active     bool
	RateLimit  int
	RateWindow int
}

type Repository interface {
	FindByKeyID(
		ctx context.Context,
		keyID string,
	) (*KeyRecord, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) FindByKeyID(
	ctx context.Context,
	keyID string,
) (*KeyRecord, error) {
	query := `
		SELECT
			k.client_id,
			k.key_hash,
			k.scopes,
			(
				c.is_active = TRUE
				AND k.revoked_at IS NULL
				AND (
					k.expires_at IS NULL
					OR k.expires_at > NOW()
				)
			) AS active,
			c.rate_limit,
			c.rate_window_seconds
		FROM api_keys k
		INNER JOIN api_clients c ON c.id = k.client_id
		WHERE k.key_id = $1
	`

	var record KeyRecord

	err := r.db.QueryRow(ctx, query, keyID).Scan(
		&record.ClientID,
		&record.KeyHash,
		&record.Scopes,
		&record.Active,
		&record.RateLimit,
		&record.RateWindow,
	)

	if err != nil {
		return nil, err
	}

	return &record, nil
}

// MarkUsed memperbarui waktu terakhir API key digunakan.
func (r *PostgresRepository) MarkUsed(
	ctx context.Context,
	keyID string,
) error {
	query := `
		UPDATE api_keys
		SET last_used_at = $2
		WHERE key_id = $1
	`

	_, err := r.db.Exec(ctx, query, keyID, time.Now().UTC())

	return err
}

// CreateKey menyimpan metadata API key baru.
func (r *PostgresRepository) CreateKey(
	ctx context.Context,
	clientID int64,
	keyID string,
	keyHash []byte,
	scopes []string,
	expiresAt *time.Time,
) (int64, error) {
	query := `
		INSERT INTO api_keys (
			client_id,
			key_id,
			key_hash,
			scopes,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int64

	err := r.db.QueryRow(
		ctx,
		query,
		clientID,
		keyID,
		keyHash,
		scopes,
		expiresAt,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

// RevokeKey menandai API key sebagai revoked.
func (r *PostgresRepository) RevokeKey(
	ctx context.Context,
	clientID int64,
	keyID string,
) (bool, error) {
	query := `
		UPDATE api_keys
		SET revoked_at = NOW()
		WHERE client_id = $1
		  AND key_id = $2
		  AND revoked_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, clientID, keyID)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() == 1, nil
}

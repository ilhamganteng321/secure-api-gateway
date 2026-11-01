package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenReuse   = errors.New("refresh token reuse detected")
	ErrRefreshUserInactive = errors.New("refresh token user inactive")
)

type RefreshRepository struct {
	db *pgxpool.Pool
}

func NewRefreshRepository(db *pgxpool.Pool) *RefreshRepository {
	return &RefreshRepository{db: db}
}

func (r *RefreshRepository) FindActiveUserID(
	ctx context.Context,
	tokenHash string,
) (int64, error) {
	var userID int64

	err := r.db.QueryRow(ctx, `
		SELECT user_id
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(&userID)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInvalidRefreshToken
	}
	if err != nil {
		return 0, fmt.Errorf("find refresh token owner: %w", err)
	}

	return userID, nil
}

func (r *RefreshRepository) CreateSession(
	ctx context.Context,
	userID int64,
	tokenHash string,
	ttl time.Duration,
) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (
			user_id, family_id, token_hash, expires_at
		)
		SELECT $1, gen_random_uuid(), $2, NOW() + ($3 * INTERVAL '1 second')
		FROM users
		WHERE id = $1 AND is_active = TRUE
	`, userID, tokenHash, ttl.Seconds())

	if err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}

	return nil
}

func (r *RefreshRepository) Rotate(
	ctx context.Context,
	oldHash string,
	newHash string,
	ttl time.Duration,
) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin refresh transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		id        int64
		userID    int64
		familyID  string
		expiresAt time.Time
		revokedAt pgtype.Timestamptz
		rotatedAt pgtype.Timestamptz
		isActive  bool
	)

	err = tx.QueryRow(ctx, `
		SELECT rt.id, rt.user_id, rt.family_id::text,
		       rt.expires_at, rt.revoked_at, rt.rotated_at,
		       u.is_active
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		WHERE rt.token_hash = $1
		FOR UPDATE OF rt
	`, oldHash).Scan(
		&id, &userID, &familyID,
		&expiresAt, &revokedAt, &rotatedAt, &isActive,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInvalidRefreshToken
	}
	if err != nil {
		return 0, fmt.Errorf("read refresh session: %w", err)
	}

	// Reuse of a token previously rotated is a security event.
	if revokedAt.Valid && rotatedAt.Valid {
		_, err = tx.Exec(ctx, `
			UPDATE refresh_tokens
			SET revoked_at = COALESCE(revoked_at, NOW())
			WHERE family_id = $1::uuid
		`, familyID)
		if err != nil {
			return 0, fmt.Errorf("revoke reused token family: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit reuse response: %w", err)
		}

		return 0, ErrRefreshTokenReuse
	}

	if revokedAt.Valid || !expiresAt.After(time.Now()) {
		return 0, ErrInvalidRefreshToken
	}

	if !isActive {
		_, err = tx.Exec(ctx, `
			UPDATE refresh_tokens
			SET revoked_at = COALESCE(revoked_at, NOW())
			WHERE family_id = $1::uuid
		`, familyID)
		if err != nil {
			return 0, fmt.Errorf("revoke inactive user sessions: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit inactive session revocation: %w", err)
		}
		return 0, ErrRefreshUserInactive
	}

	tag, err := tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), rotated_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`, id)
	if err != nil {
		return 0, fmt.Errorf("revoke previous refresh token: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return 0, ErrInvalidRefreshToken
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (
			user_id, family_id, token_hash, expires_at
		)
		VALUES ($1, $2::uuid, $3, NOW() + ($4 * INTERVAL '1 second'))
	`, userID, familyID, newHash, ttl.Seconds())
	if err != nil {
		return 0, fmt.Errorf("store replacement refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit token rotation: %w", err)
	}

	return userID, nil
}

func (r *RefreshRepository) Revoke(
	ctx context.Context,
	tokenHash string,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE token_hash = $1
	`, tokenHash)

	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

package middleware

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AccessRepository struct {
	db *pgxpool.Pool
}

func NewAccessRepository(db *pgxpool.Pool) *AccessRepository {
	return &AccessRepository{db: db}
}

func (r *AccessRepository) IsUserActive(
	ctx context.Context,
	userID int64,
) (bool, error) {
	var active bool

	err := r.db.QueryRow(ctx, `
		SELECT is_active
		FROM users
		WHERE id = $1
	`, userID).Scan(&active)

	if err != nil {
		return false, fmt.Errorf("check user status: %w", err)
	}

	return active, nil
}

func (r *AccessRepository) HasAnyRole(
	ctx context.Context,
	userID int64,
	roles []string,
) (bool, error) {
	if len(roles) == 0 {
		return false, nil
	}

	var allowed bool

	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id
			JOIN users u ON u.id = ur.user_id
			WHERE ur.user_id = $1
			  AND u.is_active = TRUE
			  AND r.name = ANY($2::text[])
		)
	`, userID, roles).Scan(&allowed)

	if err != nil {
		return false, fmt.Errorf("check user role: %w", err)
	}

	return allowed, nil
}

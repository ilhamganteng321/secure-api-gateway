package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmailExists = errors.New("email already registered")

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	IsActive     bool
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(
	ctx context.Context,
	email string,
	passwordHash string,
) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin create user transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var userID int64

	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id
	`, email, passwordHash).Scan(&userID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrEmailExists
		}
		return 0, fmt.Errorf("insert user: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id
		FROM roles
		WHERE name = 'user'
	`, userID)
	if err != nil {
		return 0, fmt.Errorf("assign default role: %w", err)
	}

	var roleCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM user_roles
		WHERE user_id = $1
	`, userID).Scan(&roleCount)
	if err != nil {
		return 0, fmt.Errorf("verify default role: %w", err)
	}
	if roleCount == 0 {
		return 0, errors.New("default user role is missing")
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit create user transaction: %w", err)
	}

	return userID, nil
}

func (r *Repository) FindByEmail(
	ctx context.Context,
	email string,
) (*User, error) {
	var user User

	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, is_active
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pgx.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return &user, nil
}

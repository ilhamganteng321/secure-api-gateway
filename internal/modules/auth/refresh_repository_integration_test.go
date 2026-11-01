//go:build integration

package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupRefreshIntegrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test PostgreSQL: %v", err)
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Fatalf("ping test PostgreSQL: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
func createRefreshTestUser(t *testing.T, db *pgxpool.Pool) int64 {
	t.Helper()

	ctx := context.Background()
	email := fmt.Sprintf(
		"refresh-test-%d@example.invalid",
		time.Now().UnixNano(),
	)

	var userID int64

	err := db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, is_active)
		VALUES ($1, 'integration-test-hash', TRUE)
		RETURNING id
	`, email).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		_, _ = db.Exec(
			cleanupCtx,
			`DELETE FROM refresh_tokens WHERE user_id = $1`,
			userID,
		)
		_, _ = db.Exec(
			cleanupCtx,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	return userID
}
func TestRefreshRotation_ReuseRevokesFamily(t *testing.T) {
	db := setupRefreshIntegrationDB(t)
	repo := NewRefreshRepository(db)
	userID := createRefreshTestUser(t, db)

	ctx := context.Background()
	ttl := time.Hour

	oldRaw, err := GenerateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	newRaw, err := GenerateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}

	oldHash := HashRefreshToken(oldRaw)
	newHash := HashRefreshToken(newRaw)

	if err := repo.CreateSession(ctx, userID, oldHash, ttl); err != nil {
		t.Fatalf("create session: %v", err)
	}

	gotUserID, err := repo.Rotate(ctx, oldHash, newHash, ttl)
	if err != nil {
		t.Fatalf("first rotation should succeed: %v", err)
	}
	if gotUserID != userID {
		t.Fatalf("got user ID %d, want %d", gotUserID, userID)
	}

	_, err = repo.Rotate(ctx, oldHash, HashRefreshToken("another-new-token"), ttl)
	if !errors.Is(err, ErrRefreshTokenReuse) {
		t.Fatalf("got error %v, want ErrRefreshTokenReuse", err)
	}

	var revokedAt *time.Time
	err = db.QueryRow(ctx, `
		SELECT revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, newHash).Scan(&revokedAt)
	if err != nil {
		t.Fatalf("query rotated token: %v", err)
	}

	if revokedAt == nil {
		t.Fatal("rotated token should be revoked after token reuse")
	}
}

func TestRefreshRotation_ConcurrentReuseRevokesFamily(t *testing.T) {
	db := setupRefreshIntegrationDB(t)
	repo := NewRefreshRepository(db)
	userID := createRefreshTestUser(t, db)

	ctx := context.Background()
	ttl := time.Hour

	oldRaw, err := GenerateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	oldHash := HashRefreshToken(oldRaw)

	if err := repo.CreateSession(ctx, userID, oldHash, ttl); err != nil {
		t.Fatalf("create session: %v", err)
	}

	newHashes := make([]string, 2)
	for i := range newHashes {
		raw, err := GenerateRefreshToken()
		if err != nil {
			t.Fatal(err)
		}
		newHashes[i] = HashRefreshToken(raw)
	}

	type result struct {
		index  int
		userID int64
		err    error
	}

	results := make(chan result, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()
			<-start

			rotationCtx, cancel := context.WithTimeout(
				context.Background(),
				10*time.Second,
			)
			defer cancel()

			id, err := repo.Rotate(
				rotationCtx,
				oldHash,
				newHashes[index],
				ttl,
			)
			results <- result{
				index:  index,
				userID: id,
				err:    err,
			}
		}(i)
	}

	close(start)
	wg.Wait()
	close(results)

	successes := 0
	reuseDetected := 0
	successfulIndex := -1

	for r := range results {
		switch {
		case r.err == nil:
			successes++
			successfulIndex = r.index

			if r.userID != userID {
				t.Errorf("got user ID %d, want %d", r.userID, userID)
			}

		case errors.Is(r.err, ErrRefreshTokenReuse):
			reuseDetected++

		default:
			t.Errorf("unexpected rotation error: %v", r.err)
		}
	}

	if successes != 1 {
		t.Errorf("got %d successful rotations, want exactly 1", successes)
	}
	if reuseDetected != 1 {
		t.Errorf("got %d reuse detections, want exactly 1", reuseDetected)
	}

	if successfulIndex >= 0 {
		var revokedAt *time.Time

		err := db.QueryRow(ctx, `
			SELECT revoked_at
			FROM refresh_tokens
			WHERE token_hash = $1
		`, newHashes[successfulIndex]).Scan(&revokedAt)

		if err != nil {
			t.Fatalf("query successful rotation token: %v", err)
		}

		if revokedAt == nil {
			t.Error("successful rotation token should be revoked after reuse detection")
		}
	}
}

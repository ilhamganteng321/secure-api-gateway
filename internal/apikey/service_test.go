package apikey_test

import (
	"context"
	"testing"
	"time"

	"github.com/ilhamganteng321/secure-api-gateway/internal/apikey"
)

type managementRepoMock struct {
	createdKeyID string
	createdHash  []byte
	revoked      bool
}

func (m *managementRepoMock) CreateKey(
	ctx context.Context,
	clientID int64,
	keyID string,
	keyHash []byte,
	scopes []string,
	expiresAt *time.Time,
) (int64, error) {
	m.createdKeyID = keyID
	m.createdHash = keyHash

	return 100, nil
}

func (m *managementRepoMock) RevokeKey(
	ctx context.Context,
	clientID int64,
	keyID string,
) (bool, error) {
	m.revoked = true
	return true, nil
}

func TestCreateAPIKey(t *testing.T) {
	repo := &managementRepoMock{}
	service := apikey.NewService(repo)

	result, err := service.Create(
		context.Background(),
		apikey.CreateKeyInput{
			ClientID: 10,
			Scopes:   []string{"products:read"},
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if result.ID != 100 {
		t.Fatalf("expected ID 100, got %d", result.ID)
	}

	if result.Plaintext == "" {
		t.Fatal("expected plaintext API key")
	}

	if repo.createdKeyID != result.KeyID {
		t.Fatal("key ID was not persisted")
	}

	if len(repo.createdHash) == 0 {
		t.Fatal("expected API key hash to be persisted")
	}
}

func TestCreateAPIKeyRejectsInvalidClient(t *testing.T) {
	service := apikey.NewService(&managementRepoMock{})

	_, err := service.Create(
		context.Background(),
		apikey.CreateKeyInput{
			ClientID: 0,
		},
	)

	if err != apikey.ErrInvalidClientID {
		t.Fatalf("expected ErrInvalidClientID, got %v", err)
	}
}

func TestCreateAPIKeyRejectsPastExpiry(t *testing.T) {
	service := apikey.NewService(&managementRepoMock{})

	expired := time.Now().Add(-time.Hour)

	_, err := service.Create(
		context.Background(),
		apikey.CreateKeyInput{
			ClientID:  10,
			ExpiresAt: &expired,
		},
	)

	if err != apikey.ErrInvalidExpiry {
		t.Fatalf("expected ErrInvalidExpiry, got %v", err)
	}
}

func TestRevokeAPIKey(t *testing.T) {
	repo := &managementRepoMock{}
	service := apikey.NewService(repo)

	revoked, err := service.Revoke(
		context.Background(),
		10,
		"7f2a91bc83d4e501",
	)

	if err != nil {
		t.Fatal(err)
	}

	if !revoked || !repo.revoked {
		t.Fatal("expected API key to be revoked")
	}
}

package apikey

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidClientID = errors.New("invalid client id")
	ErrInvalidExpiry   = errors.New("invalid expiration")
)

type CreateKeyInput struct {
	ClientID  int64
	Scopes    []string
	ExpiresAt *time.Time
}

type CreatedKey struct {
	ID        int64
	KeyID     string
	Plaintext string
	ExpiresAt *time.Time
}

type ManagementRepository interface {
	CreateKey(
		ctx context.Context,
		clientID int64,
		keyID string,
		keyHash []byte,
		scopes []string,
		expiresAt *time.Time,
	) (int64, error)

	RevokeKey(
		ctx context.Context,
		clientID int64,
		keyID string,
	) (bool, error)
}

type Service struct {
	repo ManagementRepository
}

func NewService(repo ManagementRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(
	ctx context.Context,
	input CreateKeyInput,
) (*CreatedKey, error) {
	if input.ClientID <= 0 {
		return nil, ErrInvalidClientID
	}

	if input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now()) {
		return nil, ErrInvalidExpiry
	}

	generated, err := Generate()
	if err != nil {
		return nil, err
	}

	id, err := s.repo.CreateKey(
		ctx,
		input.ClientID,
		generated.KeyID,
		generated.Hash,
		input.Scopes,
		input.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	return &CreatedKey{
		ID:        id,
		KeyID:     generated.KeyID,
		Plaintext: generated.Plaintext,
		ExpiresAt: input.ExpiresAt,
	}, nil
}

func (s *Service) Revoke(
	ctx context.Context,
	clientID int64,
	keyID string,
) (bool, error) {
	if clientID <= 0 || keyID == "" {
		return false, ErrInvalidClientID
	}

	return s.repo.RevokeKey(ctx, clientID, keyID)
}

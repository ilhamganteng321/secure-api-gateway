package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrInvalidLogin = errors.New("invalid email or password")
	ErrInactiveUser = errors.New("user account is inactive")
)

type Service struct {
	repo        *Repository
	tokens      *TokenManager
	refreshRepo *RefreshRepository
	refreshTTL  time.Duration
}

func NewService(
	repo *Repository,
	tokens *TokenManager,
	refreshRepo *RefreshRepository,
	refreshTTL time.Duration,
) *Service {
	return &Service{
		repo:        repo,
		tokens:      tokens,
		refreshRepo: refreshRepo,
		refreshTTL:  refreshTTL,
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateCredentials(email, password string) error {
	email = normalizeEmail(email)

	if len(email) == 0 || len(email) > 254 {
		return ErrInvalidInput
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return ErrInvalidInput
	}

	if !utf8.ValidString(password) ||
		utf8.RuneCountInString(password) < 12 ||
		len(password) > 128 {
		return ErrInvalidInput
	}

	return nil
}

func (s *Service) Register(
	ctx context.Context,
	email, password string,
) (int64, error) {
	if err := validateCredentials(email, password); err != nil {
		return 0, err
	}

	email = normalizeEmail(email)

	passwordHash, err := HashPassword(password)
	if err != nil {
		return 0, err
	}

	return s.repo.CreateUser(ctx, email, passwordHash)
}

func (s *Service) Login(
	ctx context.Context,
	email, password string,
) (string, string, int64, error) {
	email = normalizeEmail(email)

	if len(email) == 0 || len(email) > 254 ||
		len(password) == 0 || len(password) > 128 ||
		!utf8.ValidString(password) {
		return "", "", 0, ErrInvalidLogin
	}

	user, err := s.repo.FindByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", 0, ErrInvalidLogin
	}
	if err != nil {
		return "", "", 0, err
	}
	if !VerifyPassword(password, user.PasswordHash) {
		return "", "", 0, ErrInvalidLogin
	}
	if !user.IsActive {
		return "", "", 0, ErrInactiveUser
	}

	accessToken, err := s.tokens.GenerateAccessToken(user.ID)
	if err != nil {
		return "", "", 0, err
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return "", "", 0, err
	}

	err = s.refreshRepo.CreateSession(
		ctx,
		user.ID,
		HashRefreshToken(refreshToken),
		s.refreshTTL,
	)
	if err != nil {
		return "", "", 0, err
	}

	return accessToken, refreshToken, user.ID, nil
}

func (s *Service) Refresh(
	ctx context.Context,
	oldToken string,
) (string, string, error) {
	if len(oldToken) < 40 || len(oldToken) > 100 {
		return "", "", ErrInvalidRefreshToken
	}

	newRefreshToken, err := GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	// Generate the access token only after we know which user
	// owns the refresh token. Rotate() performs the atomic DB update.
	//
	// This implementation first uses a separate lookup helper
	// to resolve the user and family before rotation.
	userID, err := s.refreshRepo.FindActiveUserID(
		ctx,
		HashRefreshToken(oldToken),
	)
	if err != nil {
		return "", "", err
	}

	accessToken, err := s.tokens.GenerateAccessToken(userID)
	if err != nil {
		return "", "", err
	}

	rotatedUserID, err := s.refreshRepo.Rotate(
		ctx,
		HashRefreshToken(oldToken),
		HashRefreshToken(newRefreshToken),
		s.refreshTTL,
	)
	if err != nil {
		return "", "", err
	}

	if rotatedUserID != userID {
		return "", "", ErrInvalidRefreshToken
	}

	return accessToken, newRefreshToken, nil
}

func (s *Service) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	if len(refreshToken) < 40 || len(refreshToken) > 100 {
		return ErrInvalidRefreshToken
	}

	return s.refreshRepo.Revoke(
		ctx,
		HashRefreshToken(refreshToken),
	)
}

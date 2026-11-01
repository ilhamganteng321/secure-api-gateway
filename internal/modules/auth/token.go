package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
}

type AccessClaims struct {
	jwt.RegisteredClaims
}

func NewTokenManager(
	secretBase64 string,
	issuer string,
	audience string,
	ttl time.Duration,
) (*TokenManager, error) {
	secret, err := base64.StdEncoding.DecodeString(secretBase64)
	if err != nil || len(secret) < 32 {
		return nil, errors.New("JWT secret must be valid base64 containing at least 32 bytes")
	}

	if issuer == "" || audience == "" || ttl <= 0 {
		return nil, errors.New("invalid JWT configuration")
	}

	return &TokenManager{
		secret:   secret,
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
	}, nil
}

func (tm *TokenManager) GenerateAccessToken(userID int64) (string, error) {
	now := time.Now()

	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", fmt.Errorf("generate token ID: %w", err)
	}

	claims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Subject:   strconv.FormatInt(userID, 10),
			Audience:  jwt.ClaimStrings{tm.audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        base64.RawURLEncoding.EncodeToString(jtiBytes),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(tm.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signed, nil
}

func (tm *TokenManager) VerifyAccessToken(raw string) (int64, error) {
	claims, err := tm.VerifyAccessTokenClaims(raw)
	if err != nil {
		return 0, err
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, errors.New("invalid subject")
	}

	return userID, nil
}

func (tm *TokenManager) VerifyAccessTokenClaims(raw string) (*AccessClaims, error) {
	claims := &AccessClaims{}

	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing algorithm")
			}
			return tm.secret, nil
		},
		jwt.WithIssuer(tm.issuer),
		jwt.WithAudience(tm.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(0),
	)

	if err != nil || !token.Valid {
		return nil, errors.New("invalid access token")
	}

	if claims.Subject == "" {
		return nil, errors.New("missing subject")
	}

	if _, err := strconv.ParseInt(claims.Subject, 10, 64); err != nil {
		return nil, errors.New("invalid subject")
	}

	return claims, nil
}

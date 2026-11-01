package auth

import (
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecretA = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="
	testSecretB = "YWJjZGVmMDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODk="
)

func newTestTokenManager(t *testing.T, secret, issuer, audience string, ttl time.Duration) *TokenManager {
	t.Helper()

	manager, err := NewTokenManager(secret, issuer, audience, ttl)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	return manager
}

func TestVerifyAccessToken_RejectsInvalidTokens(t *testing.T) {
	manager := newTestTokenManager(
		t,
		testSecretA,
		"secure-api-gateway",
		"secure-api-client",
		15*time.Minute,
	)

	tests := []struct {
		name  string
		token string
	}{
		{name: "empty token", token: ""},
		{name: "malformed token", token: "bukan.jwt.valid"},
		{name: "missing segments", token: "header.payload"},
		{name: "random string", token: strings.Repeat("x", 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.VerifyAccessToken(tt.token)
			if err == nil {
				t.Fatal("VerifyAccessToken() should reject invalid token")
			}
		})
	}
}

func TestVerifyAccessToken_RejectsWrongSignature(t *testing.T) {
	issuer := "secure-api-gateway"
	audience := "secure-api-client"

	signer := newTestTokenManager(t, testSecretA, issuer, audience, 15*time.Minute)
	verifier := newTestTokenManager(t, testSecretB, issuer, audience, 15*time.Minute)

	token, err := signer.GenerateAccessToken(42)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	_, err = verifier.VerifyAccessToken(token)
	if err == nil {
		t.Fatal("VerifyAccessToken() should reject token with wrong signature")
	}
}

func TestVerifyAccessToken_RejectsExpiredToken(t *testing.T) {
	manager := newTestTokenManager(
		t,
		testSecretA,
		"secure-api-gateway",
		"secure-api-client",
		time.Millisecond,
	)

	token, err := manager.GenerateAccessToken(42)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	time.Sleep(25 * time.Millisecond)

	_, err = manager.VerifyAccessToken(token)
	if err == nil {
		t.Fatal("VerifyAccessToken() should reject expired token")
	}
}
func TestVerifyAccessToken_RejectsWrongIssuer(t *testing.T) {
	audience := "secure-api-client"

	signer := newTestTokenManager(
		t,
		testSecretA,
		"issuer-a",
		audience,
		15*time.Minute,
	)
	verifier := newTestTokenManager(
		t,
		testSecretA,
		"issuer-b",
		audience,
		15*time.Minute,
	)

	token, err := signer.GenerateAccessToken(42)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	_, err = verifier.VerifyAccessToken(token)
	if err == nil {
		t.Fatal("VerifyAccessToken() should reject token with wrong issuer")
	}
}

func TestVerifyAccessToken_RejectsWrongAudience(t *testing.T) {
	issuer := "secure-api-gateway"

	signer := newTestTokenManager(
		t,
		testSecretA,
		issuer,
		"audience-a",
		15*time.Minute,
	)
	verifier := newTestTokenManager(
		t,
		testSecretA,
		issuer,
		"audience-b",
		15*time.Minute,
	)

	token, err := signer.GenerateAccessToken(42)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	_, err = verifier.VerifyAccessToken(token)
	if err == nil {
		t.Fatal("VerifyAccessToken() should reject token with wrong audience")
	}
}

func TestVerifyAccessToken_RejectsMissingSubject(t *testing.T) {
	manager := newTestTokenManager(
		t,
		testSecretA,
		"secure-api-gateway",
		"secure-api-client",
		15*time.Minute,
	)

	secret, err := base64.StdEncoding.DecodeString(testSecretA)
	if err != nil {
		t.Fatal(err)
	}

	claims := jwt.MapClaims{
		"iss": "secure-api-gateway",
		"aud": []string{"secure-api-client"},
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	_, err = manager.VerifyAccessToken(signed)
	if err == nil {
		t.Fatal("VerifyAccessToken() should reject token with missing subject")
	}
}

func TestVerifyAccessToken_RejectsInvalidSubject(t *testing.T) {
	manager := newTestTokenManager(
		t,
		testSecretA,
		"secure-api-gateway",
		"secure-api-client",
		15*time.Minute,
	)

	secret, err := base64.StdEncoding.DecodeString(testSecretA)
	if err != nil {
		t.Fatal(err)
	}

	claims := jwt.MapClaims{
		"iss": "secure-api-gateway",
		"sub": "not-a-number",
		"aud": []string{"secure-api-client"},
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	_, err = manager.VerifyAccessToken(signed)
	if err == nil {
		t.Fatal("VerifyAccessToken() should reject token with invalid subject")
	}
}

func TestVerifyAccessToken_RejectsUnexpectedAlgorithm(t *testing.T) {
	manager := newTestTokenManager(
		t,
		testSecretA,
		"secure-api-gateway",
		"secure-api-client",
		15*time.Minute,
	)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"secure-api-gateway","sub":"42","aud":["secure-api-client"],"exp":` + strconv.FormatInt(time.Now().Add(15*time.Minute).Unix(), 10) + `,"iat":` + strconv.FormatInt(time.Now().Unix(), 10) + `}`))
	noneToken := header + "." + payload + "."

	_, err := manager.VerifyAccessToken(noneToken)
	if err == nil {
		t.Fatal("VerifyAccessToken() should reject token with unexpected algorithm")
	}
}

func TestGenerateAccessToken_RoundTrip(t *testing.T) {
	manager := newTestTokenManager(
		t,
		testSecretA,
		"secure-api-gateway",
		"secure-api-client",
		15*time.Minute,
	)

	token, err := manager.GenerateAccessToken(123)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	userID, err := manager.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("VerifyAccessToken() error = %v", err)
	}
	if userID != 123 {
		t.Fatalf("userID = %d, want 123", userID)
	}
}

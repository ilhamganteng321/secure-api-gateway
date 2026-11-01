package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

type GeneratedKey struct {
	Plaintext string
	KeyID     string
	Hash      []byte
}

func Generate() (*GeneratedKey, error) {
	idBytes := make([]byte, 8)
	secretBytes := make([]byte, 32)

	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("generate key id: %w", err)
	}

	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}

	keyID := hex.EncodeToString(idBytes)
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)

	plaintext := "sk_live_" + keyID + "." + secret

	hash := sha256.Sum256([]byte(secret))

	return &GeneratedKey{
		Plaintext: plaintext,
		KeyID:     keyID,
		Hash:      hash[:],
	}, nil
}

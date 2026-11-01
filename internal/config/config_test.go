package config

import (
	"encoding/base64"
	"testing"
)

func setValidEnv(t *testing.T) {
	t.Helper()

	values := map[string]string{
		"APP_NAME":          "secure-api-gateway",
		"APP_ENV":           "test",
		"APP_PORT":          "3000",
		"DATABASE_URL":      "postgres://user:pass@localhost:5432/gateway_test?sslmode=disable",
		"REDIS_ADDR":        "localhost:6379",
		"REDIS_PASSWORD":    "",
		"REDIS_DB":          "15",
		"JWT_ISSUER":        "secure-api-gateway",
		"JWT_AUDIENCE":      "secure-api-client",
		"JWT_SECRET":        base64.StdEncoding.EncodeToString(make([]byte, 32)),
		"JWT_ACCESS_TTL":    "15m",
		"REFRESH_TOKEN_TTL": "168h",
	}

	for key, value := range values {
		t.Setenv(key, value)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppPort != "3000" {
		t.Fatalf("AppPort = %q, want 3000", cfg.AppPort)
	}

	if cfg.RedisDB != 15 {
		t.Fatalf("RedisDB = %d, want 15", cfg.RedisDB)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_PORT", "99999")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() seharusnya menolak port yang tidak valid")
	}
}

func TestLoad_InvalidJWTSecret(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_SECRET", base64.StdEncoding.EncodeToString([]byte("pendek")))

	_, err := Load()
	if err == nil {
		t.Fatal("Load() seharusnya menolak JWT_SECRET yang terlalu pendek")
	}
}

func TestLoad_InvalidRedisDB(t *testing.T) {
	setValidEnv(t)
	t.Setenv("REDIS_DB", "bukan-angka")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() seharusnya menolak REDIS_DB non-numerik")
	}
}

func TestValidate_RejectsInvalidDatabaseURL(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	cfg.DatabaseURL = "http://localhost:5432/gateway"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() seharusnya menolak URL database non-PostgreSQL")
	}
}

func TestLoad_RequiresJWTIssuer(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_ISSUER", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() seharusnya menolak JWT_ISSUER kosong")
	}
}

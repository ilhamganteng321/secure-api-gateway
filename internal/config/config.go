package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName string
	AppEnv  string
	AppPort string

	DatabaseURL string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTIssuer   string
	JWTAudience string
	JWTSecret   string

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	DBMaxConns        int
	DBMinConns        int
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration

	OTELServiceName      string
	OTELExporterEndpoint string
	OTELTracesSamplerArg float64
}

func Load() (Config, error) {
	var cfg Config
	var err error

	cfg.AppName = getEnv("APP_NAME", "secure-api-gateway")
	cfg.AppEnv = getEnv("APP_ENV", "development")
	cfg.AppPort = getEnv("APP_PORT", "3000")

	cfg.DatabaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))

	cfg.RedisAddr = getEnv("REDIS_ADDR", "localhost:6379")
	cfg.RedisPassword = os.Getenv("REDIS_PASSWORD")
	cfg.RedisDB, err = strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return cfg, errors.New("REDIS_DB harus berupa angka")
	}

	cfg.JWTIssuer = strings.TrimSpace(os.Getenv("JWT_ISSUER"))
	cfg.JWTAudience = strings.TrimSpace(os.Getenv("JWT_AUDIENCE"))
	cfg.JWTSecret = strings.TrimSpace(os.Getenv("JWT_SECRET"))

	cfg.AccessTokenTTL, err = time.ParseDuration(
		getEnv("JWT_ACCESS_TTL", "15m"),
	)
	if err != nil {
		return cfg, fmt.Errorf("JWT_ACCESS_TTL tidak valid: %w", err)
	}

	cfg.RefreshTokenTTL, err = time.ParseDuration(
		getEnv("REFRESH_TOKEN_TTL", "168h"),
	)
	if err != nil {
		return cfg, fmt.Errorf("REFRESH_TOKEN_TTL tidak valid: %w", err)
	}

	cfg.DBMaxConns, err = strconv.Atoi(getEnv("DB_MAX_CONNS", "20"))
	if err != nil {
		return cfg, errors.New("DB_MAX_CONNS harus berupa angka")
	}

	cfg.DBMinConns, err = strconv.Atoi(getEnv("DB_MIN_CONNS", "2"))
	if err != nil {
		return cfg, errors.New("DB_MIN_CONNS harus berupa angka")
	}

	cfg.DBMaxConnLifetime, err = time.ParseDuration(getEnv("DB_MAX_CONN_LIFETIME", "30m"))
	if err != nil {
		return cfg, fmt.Errorf("DB_MAX_CONN_LIFETIME tidak valid: %w", err)
	}

	cfg.DBMaxConnIdleTime, err = time.ParseDuration(getEnv("DB_MAX_CONN_IDLE_TIME", "5m"))
	if err != nil {
		return cfg, fmt.Errorf("DB_MAX_CONN_IDLE_TIME tidak valid: %w", err)
	}

	cfg.OTELServiceName = getEnv("OTEL_SERVICE_NAME", cfg.AppName)
	cfg.OTELExporterEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")

	sampling, err := strconv.ParseFloat(getEnv("OTEL_TRACES_SAMPLER_ARG", "1.0"), 64)
	if err != nil {
		return cfg, fmt.Errorf("OTEL_TRACES_SAMPLER_ARG tidak valid: %w", err)
	}
	cfg.OTELTracesSamplerArg = sampling

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	if err := cfg.ValidateProduction(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.AppName) == "" {
		return errors.New("APP_NAME wajib diisi")
	}

	switch c.AppEnv {
	case "development", "test", "production":
	default:
		return errors.New(
			"APP_ENV harus development, test, atau production",
		)
	}

	port, err := strconv.Atoi(c.AppPort)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("APP_PORT harus berada di antara 1 dan 65535")
	}

	dbURL, err := url.Parse(c.DatabaseURL)
	if err != nil ||
		(dbURL.Scheme != "postgres" && dbURL.Scheme != "postgresql") ||
		dbURL.Host == "" ||
		dbURL.Path == "" ||
		dbURL.Path == "/" {
		return errors.New("DATABASE_URL tidak valid")
	}

	if _, _, err := net.SplitHostPort(c.RedisAddr); err != nil {
		return errors.New("REDIS_ADDR harus berbentuk host:port")
	}

	if c.RedisDB < 0 {
		return errors.New("REDIS_DB tidak boleh negatif")
	}

	if c.JWTIssuer == "" {
		return errors.New("JWT_ISSUER wajib diisi")
	}

	if c.JWTAudience == "" {
		return errors.New("JWT_AUDIENCE wajib diisi")
	}

	secret, err := base64.StdEncoding.DecodeString(c.JWTSecret)
	if err != nil || len(secret) < 32 {
		return errors.New(
			"JWT_SECRET harus berupa Base64 valid dengan minimal 32 byte hasil decode",
		)
	}

	if c.AccessTokenTTL <= 0 {
		return errors.New("JWT_ACCESS_TTL harus lebih dari 0")
	}

	if c.RefreshTokenTTL <= 0 {
		return errors.New("REFRESH_TOKEN_TTL harus lebih dari 0")
	}

	if c.AppEnv == "production" && c.JWTSecret == "" {
		return errors.New("JWT_SECRET wajib diisi untuk production")
	}

	if c.OTELServiceName == "" {
		return errors.New("OTEL_SERVICE_NAME wajib diisi")
	}

	if c.OTELExporterEndpoint == "" {
		return errors.New("OTEL_EXPORTER_OTLP_ENDPOINT wajib diisi")
	}

	if c.OTELTracesSamplerArg <= 0 || c.OTELTracesSamplerArg > 1 {
		return errors.New("OTEL_TRACES_SAMPLER_ARG harus berada di antara 0 dan 1")
	}

	if c.DBMaxConns <= 0 {
		return errors.New("DB_MAX_CONNS harus lebih dari 0")
	}

	if c.DBMinConns < 0 || c.DBMinConns > c.DBMaxConns {
		return errors.New("DB_MIN_CONNS tidak valid")
	}

	if c.DBMaxConnLifetime <= 0 {
		return errors.New("DB_MAX_CONN_LIFETIME harus lebih dari 0")
	}

	if c.DBMaxConnIdleTime <= 0 {
		return errors.New("DB_MAX_CONN_IDLE_TIME harus lebih dari 0")
	}

	return nil
}

// ValidateProduction applies stricter checks for production deployments.
func (c Config) ValidateProduction() error {
	if c.AppEnv != "production" {
		return nil
	}

	if c.JWTSecret == "" || len(c.JWTSecret) < 32 {
		return errors.New("production JWT secret is too weak")
	}

	if c.DatabaseURL == "" {
		return errors.New("database URL is required")
	}

	if c.RedisAddr == "" {
		return errors.New("Redis address is required")
	}

	return nil
}
func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return strings.TrimSpace(value)
}

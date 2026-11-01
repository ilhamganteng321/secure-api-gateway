package main

import (
	"context"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/ilhamganteng321/secure-api-gateway/internal/app"
	"github.com/ilhamganteng321/secure-api-gateway/internal/config"
	"github.com/ilhamganteng321/secure-api-gateway/internal/observability"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Info("File .env tidak ditemukan; menggunakan environment sistem")
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Konfigurasi aplikasi tidak valid", "error", err)
		os.Exit(1)
	}

	startupCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		cancel()
		slog.Error("PostgreSQL configuration invalid", "error", err)
		os.Exit(1)
	}

	poolConfig.MaxConns = int32(cfg.DBMaxConns)
	poolConfig.MinConns = int32(cfg.DBMinConns)
	poolConfig.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.DBMaxConnIdleTime

	db, err := pgxpool.NewWithConfig(startupCtx, poolConfig)
	if err != nil {
		cancel()
		slog.Error("PostgreSQL initialization failed", "error", err)
		os.Exit(1)
	}

	if err := db.Ping(startupCtx); err != nil {
		cancel()
		db.Close()
		slog.Error("PostgreSQL unavailable", "error", err)
		os.Exit(1)
	}
	cancel()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	redisCtx, redisCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	err = redisClient.Ping(redisCtx).Err()
	redisCancel()

	if err != nil {
		redisClient.Close()
		db.Close()
		slog.Error("Redis unavailable", "error", err)
		os.Exit(1)
	}

	shutdownTracing, err := observability.InitTracing(
		context.Background(),
		observability.TracingConfig{
			ServiceName:   cfg.OTELServiceName,
			Endpoint:      cfg.OTELExporterEndpoint,
			SamplingRatio: cfg.OTELTracesSamplerArg,
		},
	)
	if err != nil {
		slog.Error("OpenTelemetry tracing initialization failed", "error", err)
		redisClient.Close()
		db.Close()
		os.Exit(1)
	}

	application := app.New(app.Dependencies{
		Config: cfg,
		DB:     db,
		Redis:  redisClient,
		Logger: observability.NewLogger(cfg.AppEnv),
	})

	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	if cfg.AppEnv != "production" {
		go func() {
			slog.Info("pprof server listening", "addr", "127.0.0.1:6060")
			if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
				slog.Error("pprof server stopped", "error", err)
			}
		}()
	}
	observability.StartRuntimeMonitor(monitorCtx, observability.NewLogger(cfg.AppEnv), 30*time.Second)

	slog.Info("Secure API Gateway starting", "port", cfg.AppPort)

	closeDependencies := func() {
		monitorCancel()

		redisClient.Close()
		db.Close()

		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer shutdownCancel()

		if err := shutdownTracing(shutdownCtx); err != nil {
			slog.Error("Tracing shutdown failed", "error", err)
		}
	}

	err = runServer(application, cfg.AppPort, closeDependencies)

	if err != nil {
		slog.Error("Server berhenti karena error", "error", err)
		os.Exit(1)
	}
}

func runServer(
	app *fiber.App,
	port string,
	closeDependencies func(),
) error {
	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- app.Listen(":" + port)
	}()

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil {
			return err
		}
		return nil

	case <-signalCtx.Done():
		slog.Info("Shutdown diterima; menghentikan server")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			slog.Error("Graceful shutdown gagal", "error", err)
			return err
		}

		closeDependencies()
		slog.Info("Server berhasil dihentikan")
		return nil
	}
}

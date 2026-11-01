package app

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ilhamganteng321/secure-api-gateway/internal/apikey"
	"github.com/ilhamganteng321/secure-api-gateway/internal/config"
	"github.com/ilhamganteng321/secure-api-gateway/internal/docs"
	"github.com/ilhamganteng321/secure-api-gateway/internal/gateway"
	"github.com/ilhamganteng321/secure-api-gateway/internal/health"
	"github.com/ilhamganteng321/secure-api-gateway/internal/middleware"
	"github.com/ilhamganteng321/secure-api-gateway/internal/modules/audit"
	"github.com/ilhamganteng321/secure-api-gateway/internal/modules/auth"
	"github.com/ilhamganteng321/secure-api-gateway/internal/observability"
)

type Dependencies struct {
	Config config.Config
	DB     *pgxpool.Pool
	Redis  *redis.Client
	Logger *slog.Logger
}

func New(deps Dependencies) *fiber.App {
	cfg := deps.Config
	db := deps.DB
	redisClient := deps.Redis
	logger := deps.Logger
	if logger == nil {
		logger = observability.NewLogger(cfg.AppEnv)
	}
	slog.SetDefault(logger)

	tokenManager, err := auth.NewTokenManager(
		cfg.JWTSecret,
		cfg.JWTIssuer,
		cfg.JWTAudience,
		cfg.AccessTokenTTL,
	)
	if err != nil {
		panic(err)
	}

	refreshTTL := cfg.RefreshTokenTTL

	generalLimiter := middleware.NewRateLimiter(
		redisClient,
		100,
		time.Minute,
		"secure-api:rate:general",
	)

	loginLimiter := middleware.NewRateLimiter(
		redisClient,
		5,
		time.Minute,
		"secure-api:rate:login",
	)

	auditRepo := audit.NewRepository(db)
	authRepo := auth.NewRepository(db)
	refreshRepo := auth.NewRefreshRepository(db)

	authService := auth.NewService(
		authRepo,
		tokenManager,
		refreshRepo,
		refreshTTL,
	)

	authHandler := auth.NewHandler(authService, auditRepo)

	metrics := observability.NewMetrics()
	gatewayMetrics := observability.NewGatewayMetrics(metrics.Registry)
	generalLimiter.SetMetrics(gatewayMetrics)
	loginLimiter.SetMetrics(gatewayMetrics)

	accessRepo := middleware.NewAccessRepository(db)
	authentication := middleware.NewAuthentication(
		tokenManager,
		accessRepo,
		auditRepo,
	)

	apiKeyRepo := apikey.NewPostgresRepository(db)
	apiKeyService := apikey.NewService(apiKeyRepo)
	apiKeyHandler := apikey.NewHandler(apiKeyService)

	proxy := gateway.NewProxy()
	proxy.SetGatewayMetrics(gatewayMetrics)

	upstreams := []struct {
		name    string
		rawURL  string
		timeout time.Duration
	}{
		{"users", "http://localhost:4001", 5 * time.Second},
		{"products", "http://localhost:4002", 5 * time.Second},
		{"orders", "http://localhost:4003", 5 * time.Second},
	}

	for _, item := range upstreams {
		upstream, err := gateway.NewUpstream(
			item.name,
			item.rawURL,
			item.timeout,
		)
		if err != nil {
			panic(err)
		}

		if err := proxy.Register(upstream); err != nil {
			panic(err)
		}
	}

	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		BodyLimit:    1 * 1024 * 1024,
		TrustProxy:   false,
	})

	app.Use(middleware.SecurityHeaders())
	app.Use(middleware.CORS([]string{
		"http://localhost:5173",
	}))

	app.Get("/metrics", metrics.Handler())

	app.Use(observability.TracingMiddleware())
	app.Use(observability.RequestID())
	app.Use(observability.MetricsMiddleware(metrics))
	app.Use(observability.RequestLogger(logger))
	app.Use(observability.GatewayMetricsMiddleware(gatewayMetrics))

	app.Use("/api/v1", generalLimiter.Middleware())

	app.Get("/api/v1/me", authentication.Required(), func(c fiber.Ctx) error {
		userID, ok := middleware.UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "authentication_required",
			})
		}
		return c.JSON(fiber.Map{"user_id": userID})
	})

	app.Get("/api/v1/admin", authentication.Required(), middleware.RequireRole(accessRepo, "admin"), func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Welcome to the admin area"})
	})

	app.Get("/openapi.json", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/json; charset=utf-8")
		return c.SendString(docs.OpenAPISpec)
	})

	app.Get("/scalar", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		return c.SendString(docs.ScalarHTML)
	})

	healthChecker := &health.Checker{
		DB:    db,
		Redis: redisClient,
	}
	healthHandler := health.NewHandler(healthChecker, 2*time.Second)
	app.Get("/health/live", healthHandler.Live)
	app.Get("/health/ready", healthHandler.Ready)

	authHandler.RegisterRoutes(app, loginLimiter.Middleware())

	clientAPI := app.Group("/api/v1/client", apikey.Middleware(apiKeyRepo), apikey.ClientRateLimit(redisClient, gatewayMetrics))
	clientAPI.Get("/products", apikey.RequireScope("products:read"), func(c fiber.Ctx) error {
		clientID, _ := c.Locals("api_client_id").(int64)
		return c.JSON(fiber.Map{
			"message":   "API key authentication successful",
			"client_id": clientID,
		})
	})

	adminAPIKeys := app.Group("/api/v1/admin/api-keys", authentication.Required(), middleware.RequireRole(accessRepo, "admin"))
	adminAPIKeys.Post("/", apiKeyHandler.Create)
	adminAPIKeys.Post("/revoke", apiKeyHandler.Revoke)

	app.Use("/gateway", apikey.Middleware(apiKeyRepo), apikey.ClientRateLimit(redisClient, gatewayMetrics), gateway.NewRequestPolicy().Middleware())
	gateway.RegisterRoutes(app, proxy)

	if cfg.AppEnv != "production" {
		app.Get("/benchmark", func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"status":  "ok",
				"service": "secure-api-gateway",
			})
		})
	}

	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "secure-api-gateway",
			"status":  "running",
		})
	})

	return app
}

package auth

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/ilhamganteng321/secure-api-gateway/internal/modules/audit"
)

type Handler struct {
	service *Service
	audit   audit.Recorder
}

func NewHandler(service *Service, recorder audit.Recorder) *Handler {
	return &Handler{
		service: service,
		audit:   recorder,
	}
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(c fiber.Ctx) error {
	var req refreshRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_request",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	accessToken, refreshToken, err := h.service.Refresh(
		ctx,
		req.RefreshToken,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrRefreshTokenReuse):
			audit.RecordRequest(
				c,
				h.audit,
				"auth.refresh_reuse_detected",
				"denied",
				fiber.StatusUnauthorized,
				nil,
				map[string]any{"reason": "rotated_token_reused"},
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "refresh_token_reuse_detected",
			})
		case errors.Is(err, ErrInvalidRefreshToken),
			errors.Is(err, ErrRefreshUserInactive):
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid_refresh_token",
			})
		default:
			slog.Error("refresh failed", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal_server_error",
			})
		}
	}

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

func (h *Handler) Logout(c fiber.Ctx) error {
	var req refreshRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_request",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	err := h.service.Logout(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid_refresh_token",
			})
		}

		slog.Error("logout failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal_server_error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "logout successful",
	})
}

func (h *Handler) RegisterRoutes(
	app *fiber.App,
	loginLimiter fiber.Handler,
) {
	app.Post("/api/v1/auth/register", h.Register)
	app.Post(
		"/api/v1/auth/login",
		loginLimiter,
		h.Login,
	)
	app.Post("/api/v1/auth/refresh", h.Refresh)
	app.Post("/api/v1/auth/logout", h.Logout)
}

func (h *Handler) Register(c fiber.Ctx) error {
	var req credentialsRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_request",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	userID, err := h.service.Register(ctx, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_input",
				"message": "Email atau password tidak memenuhi ketentuan.",
			})
		case errors.Is(err, ErrEmailExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "email_already_registered",
			})
		default:
			slog.Error("registration failed", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal_server_error",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "registration successful",
		"user": fiber.Map{
			"id":    userID,
			"email": strings.ToLower(strings.TrimSpace(req.Email)),
		},
	})
}

func (h *Handler) Login(c fiber.Ctx) error {
	var req credentialsRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_request",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	accessToken, refreshToken, userID, err := h.service.Login(
		ctx,
		req.Email,
		req.Password,
	)
	if err != nil {
		if errors.Is(err, ErrInvalidLogin) {
			audit.RecordRequest(
				c,
				h.audit,
				"auth.login",
				"failure",
				fiber.StatusUnauthorized,
				nil,
				map[string]any{"reason": "invalid_credentials"},
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid_credentials",
			})
		}

		if errors.Is(err, ErrInactiveUser) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "account_inactive",
			})
		}

		slog.Error("login failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal_server_error",
		})
	}

	audit.RecordRequest(
		c,
		h.audit,
		"auth.login",
		"success",
		fiber.StatusOK,
		&userID,
		nil,
	)

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
		"user": fiber.Map{
			"id": userID,
		},
	})
}

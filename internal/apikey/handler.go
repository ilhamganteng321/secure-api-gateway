package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type createKeyRequest struct {
	ClientID  int64      `json:"client_id"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type revokeKeyRequest struct {
	ClientID int64  `json:"client_id"`
	KeyID    string `json:"key_id"`
}

func (h *Handler) Create(c fiber.Ctx) error {
	var input createKeyRequest

	if err := c.Bind().Body(&input); err != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			"invalid request body",
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	result, err := h.service.Create(ctx, CreateKeyInput{
		ClientID:  input.ClientID,
		Scopes:    input.Scopes,
		ExpiresAt: input.ExpiresAt,
	})

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidClientID),
			errors.Is(err, ErrInvalidExpiry):
			return fiber.NewError(
				fiber.StatusBadRequest,
				err.Error(),
			)
		default:
			return fiber.NewError(
				fiber.StatusInternalServerError,
				"failed to create api key",
			)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         result.ID,
		"key_id":     result.KeyID,
		"api_key":    result.Plaintext,
		"expires_at": result.ExpiresAt,
		"warning":    "Copy this API key now. It will not be shown again.",
	})
}

func (h *Handler) Revoke(c fiber.Ctx) error {
	var input revokeKeyRequest

	if err := c.Bind().Body(&input); err != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			"invalid request body",
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	revoked, err := h.service.Revoke(
		ctx,
		input.ClientID,
		input.KeyID,
	)

	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			"failed to revoke api key",
		)
	}

	if !revoked {
		return fiber.NewError(
			fiber.StatusNotFound,
			"api key not found or already revoked",
		)
	}

	return c.JSON(fiber.Map{
		"message": "api key revoked successfully",
	})
}

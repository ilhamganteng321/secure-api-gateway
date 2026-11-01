package health

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	checker *Checker
	timeout time.Duration
}

func NewHandler(
	checker *Checker,
	timeout time.Duration,
) *Handler {
	return &Handler{
		checker: checker,
		timeout: timeout,
	}
}

func (h *Handler) Live(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "alive",
	})
}

func (h *Handler) Ready(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		h.timeout,
	)
	defer cancel()

	result := h.checker.Check(ctx)

	if result.Status != "ok" {
		return c.Status(
			fiber.StatusServiceUnavailable,
		).JSON(result)
	}

	return c.JSON(result)
}

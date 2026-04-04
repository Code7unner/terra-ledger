package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type ConsentHandler struct {
	repo repository.ConsentRepo
}

func NewConsentHandler(repo repository.ConsentRepo) *ConsentHandler {
	return &ConsentHandler{repo: repo}
}

func (h *ConsentHandler) Grant(c *fiber.Ctx) error {
	var input struct {
		WalletAddress string `json:"wallet_address"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if input.WalletAddress == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "wallet_address required"})
	}

	consent, err := h.repo.Grant(c.Context(), input.WalletAddress)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to grant consent"})
	}

	return c.JSON(consent)
}

func (h *ConsentHandler) Revoke(c *fiber.Ctx) error {
	var input struct {
		WalletAddress string `json:"wallet_address"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if input.WalletAddress == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "wallet_address required"})
	}

	consent, err := h.repo.Revoke(c.Context(), input.WalletAddress)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "consent not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to revoke consent"})
	}

	return c.JSON(consent)
}

func (h *ConsentHandler) Get(c *fiber.Ctx) error {
	wallet := c.Params("wallet")

	consent, err := h.repo.GetByWallet(c.Context(), wallet)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "consent not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch consent"})
	}

	return c.JSON(consent)
}

func (h *ConsentHandler) ListAccessLog(c *fiber.Ctx) error {
	wallet := c.Params("wallet")

	entries, err := h.repo.ListAccessLog(c.Context(), wallet)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list access log"})
	}

	return c.JSON(entries)
}

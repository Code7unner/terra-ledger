package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type LienHandler struct {
	lienRepo   repository.LienRepo
	parcelRepo repository.ParcelRepo
	solana     repository.SolanaClient
	logger     *zerolog.Logger
}

func NewLienHandler(lr repository.LienRepo, pr repository.ParcelRepo, solana repository.SolanaClient, logger *zerolog.Logger) *LienHandler {
	return &LienHandler{lienRepo: lr, parcelRepo: pr, solana: solana, logger: logger}
}

func (h *LienHandler) Register(c *fiber.Ctx) error {
	var input entity.RegisterLienInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	existing, err := h.lienRepo.GetActive(c.Context(), input.CadastralNumber)
	if err != nil && !errors.Is(err, entity.ErrNotFound) {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to check liens"})
	}
	if existing != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":         entity.ErrDoublePledge.Error(),
			"existing_lien": existing,
		})
	}

	parcel, err := h.parcelRepo.GetByCadastral(c.Context(), input.CadastralNumber)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "parcel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch parcel"})
	}

	lien := &entity.Encumbrance{
		ParcelID:        parcel.ID,
		CadastralNumber: input.CadastralNumber,
		LenderWallet:    input.LenderWallet,
		AmountTenge:     input.AmountTenge,
		NotaryCertHash:  input.NotaryCertHash,
		Status:          entity.LienStatusActive,
	}

	if err := h.lienRepo.Create(c.Context(), lien); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to register lien"})
	}

	return c.Status(fiber.StatusCreated).JSON(lien)
}

func (h *LienHandler) Release(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.lienRepo.UpdateStatus(c.Context(), id, entity.LienStatusReleased); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "lien not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to release lien"})
	}

	return c.JSON(fiber.Map{"status": "released"})
}

func (h *LienHandler) ListByParcel(c *fiber.Ctx) error {
	cadastral := c.Params("cadastral")

	liens, err := h.lienRepo.ListByParcel(c.Context(), cadastral)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list liens"})
	}
	if liens == nil {
		liens = []entity.Encumbrance{}
	}

	return c.JSON(liens)
}

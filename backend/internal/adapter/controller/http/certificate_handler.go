package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type CertificateHandler struct {
	certRepo   repository.CertificateRepo
	parcelRepo repository.ParcelRepo
}

func NewCertificateHandler(cr repository.CertificateRepo, pr repository.ParcelRepo) *CertificateHandler {
	return &CertificateHandler{certRepo: cr, parcelRepo: pr}
}

func (h *CertificateHandler) Mint(c *fiber.Ctx) error {
	cadastral := c.Params("cadastral")

	var input struct {
		Season  string  `json:"season"`
		NDVI    float64 `json:"ndvi_score"`
		Crop    string  `json:"crop_type"`
		Yield   float64 `json:"yield_t_ha"`
		SceneID string  `json:"sentinel_scene_id"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	parcel, err := h.parcelRepo.GetByCadastral(c.Context(), cadastral)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "parcel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch parcel"})
	}

	cert := &entity.NDVICertificate{
		ParcelID:        parcel.ID,
		CadastralNumber: cadastral,
		Season:          input.Season,
		NDVIScore:       input.NDVI,
		CropType:        input.Crop,
		YieldTHa:        input.Yield,
		SentinelSceneID: input.SceneID,
	}

	if err := h.certRepo.Create(c.Context(), cert); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mint certificate"})
	}

	return c.Status(fiber.StatusCreated).JSON(cert)
}

func (h *CertificateHandler) List(c *fiber.Ctx) error {
	cadastral := c.Params("cadastral")

	certs, err := h.certRepo.ListByParcel(c.Context(), cadastral)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list certificates"})
	}

	return c.JSON(certs)
}

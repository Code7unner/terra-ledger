package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/ndvi"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type ParcelHandler struct {
	repo   repository.ParcelRepo
	solana repository.SolanaClient
	ndvi   repository.NDVIProvider
	logger *zerolog.Logger
}

func NewParcelHandler(repo repository.ParcelRepo, solana repository.SolanaClient, ndvi repository.NDVIProvider, logger *zerolog.Logger) *ParcelHandler {
	return &ParcelHandler{repo: repo, solana: solana, ndvi: ndvi, logger: logger}
}

func (h *ParcelHandler) Register(c *fiber.Ctx) error {
	var input entity.RegisterParcelInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	parcel := &entity.Parcel{
		CadastralNumber: input.CadastralNumber,
		OwnerWallet:     input.OwnerWallet,
		AreaHa:          input.AreaHa,
		LandClass:       input.LandClass,
		Oblast:          input.Oblast,
		Rayon:           input.Rayon,
		HolderName:      input.HolderName,
		HolderIINHash:   input.HolderIINHash,
		KYCVerified:     true, // Mock KYC
	}

	if err := h.repo.Create(c.Context(), parcel); err != nil {
		if errors.Is(err, entity.ErrAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to register parcel"})
	}

	return c.Status(fiber.StatusCreated).JSON(parcel)
}

func (h *ParcelHandler) Get(c *fiber.Ctx) error {
	cadastral := c.Params("cadastral")

	parcel, err := h.repo.GetByCadastral(c.Context(), cadastral)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "parcel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch parcel"})
	}

	// Enrich with on-chain data if available
	h.enrichParcelFromChain(c, parcel)

	return c.JSON(parcel)
}

func (h *ParcelHandler) enrichParcelFromChain(c *fiber.Ctx, parcel *entity.Parcel) {
	if parcel.OnChainAddress == "" || h.solana == nil {
		return
	}

	data, err := h.solana.GetAccountInfo(c.Context(), parcel.OnChainAddress)
	if err != nil {
		h.logger.Debug().Err(err).Str("address", parcel.OnChainAddress).Msg("failed to fetch on-chain parcel data")
		return
	}

	if len(data) > 0 {
		h.logger.Debug().
			Str("cadastral", parcel.CadastralNumber).
			Int("bytes", len(data)).
			Msg("fetched on-chain parcel data")
	}
}

func (h *ParcelHandler) GetNDVI(c *fiber.Ctx) error {
	cadastral := c.Params("cadastral")

	_, err := h.repo.GetByCadastral(c.Context(), cadastral)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "parcel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch parcel"})
	}

	if h.ndvi == nil {
		return c.JSON(fiber.Map{"cadastral": cadastral, "ndvi": 0.72, "source": "default"})
	}

	lat, lon := ndvi.CentroidFor(cadastral)
	now := time.Now()
	startDate := now.AddDate(0, -1, 0).Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	score, err := h.ndvi.FetchNDVI(c.Context(), cadastral, lat, lon, startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "ndvi fetch failed"})
	}

	return c.JSON(fiber.Map{
		"cadastral":  cadastral,
		"ndvi":       score,
		"lat":        lat,
		"lon":        lon,
		"start_date": startDate,
		"end_date":   endDate,
		"source":     "copernicus",
	})
}

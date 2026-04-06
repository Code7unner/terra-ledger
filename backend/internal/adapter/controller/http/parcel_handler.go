package http

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/ndvi"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type ParcelHandler struct {
	repo      repository.ParcelRepo
	solana    repository.SolanaClient
	satellite repository.SatelliteProvider
	certRepo  repository.CertificateRepo
	geocoder  repository.Geocoder
	logger    *zerolog.Logger
}

func NewParcelHandler(
	repo repository.ParcelRepo,
	solana repository.SolanaClient,
	satellite repository.SatelliteProvider,
	certRepo repository.CertificateRepo,
	geocoder repository.Geocoder,
	logger *zerolog.Logger,
) *ParcelHandler {
	return &ParcelHandler{
		repo:      repo,
		solana:    solana,
		satellite: satellite,
		certRepo:  certRepo,
		geocoder:  geocoder,
		logger:    logger,
	}
}

func (h *ParcelHandler) List(c *fiber.Ctx) error {
	parcels, err := h.repo.ListAll(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list parcels"})
	}

	return c.JSON(fiber.Map{"parcels": parcels})
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
		KYCVerified:     true,
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

	parcel, err := h.repo.GetByCadastral(c.Context(), cadastral)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "parcel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch parcel"})
	}

	if h.satellite == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "satellite provider not configured"})
	}

	lat, lon := h.geocoder.Resolve(cadastral, parcel.Oblast)
	now := time.Now()
	startDate := now.AddDate(0, -1, 0).Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	score, err := h.satellite.FetchNDVI(c.Context(), cadastral, lat, lon, startDate, endDate)
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

const defaultTimeSeriesMonths = 12

func (h *ParcelHandler) GetSatelliteTimeSeries(c *fiber.Ctx) error {
	cadastral := c.Params("cadastral")

	parcel, err := h.repo.GetByCadastral(c.Context(), cadastral)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "parcel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch parcel"})
	}

	months := parseMonthsParam(c)
	now := time.Now()
	from := now.AddDate(0, -months, 0)

	return h.respondTimeSeries(c, cadastral, parcel, from, now, months)
}

func parseMonthsParam(c *fiber.Ctx) int {
	monthsStr := c.Query("months", "12")
	months, err := strconv.Atoi(monthsStr)
	if err != nil || months < 1 || months > 60 {
		return defaultTimeSeriesMonths
	}
	return months
}

func (h *ParcelHandler) respondTimeSeries(
	c *fiber.Ctx,
	cadastral string,
	parcel *entity.Parcel,
	from, to time.Time,
	months int,
) error {
	// Try DB first.
	if h.certRepo != nil {
		certs, err := h.certRepo.ListByParcelInRange(c.Context(), cadastral, from, to)
		if err == nil && len(certs) > 0 {
			return c.JSON(fiber.Map{
				"cadastral":    cadastral,
				"certificates": certs,
				"source":       "database",
			})
		}
	}

	// Fallback to live fetch.
	if h.satellite == nil {
		return c.JSON(fiber.Map{"cadastral": cadastral, "certificates": []any{}, "source": "none"})
	}

	lat, lon := h.geocoder.Resolve(cadastral, parcel.Oblast)
	startDate := from.Format("2006-01-02")
	endDate := to.Format("2006-01-02")

	ts, err := h.satellite.FetchTimeSeries(c.Context(), cadastral, lat, lon, startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "satellite fetch failed"})
	}

	// Persist live-fetched data so subsequent calls + credit scoring use DB.
	certs := h.persistTimeSeries(c, cadastral, parcel, ts)

	return c.JSON(fiber.Map{
		"cadastral":    cadastral,
		"intervals":    ts.Intervals,
		"certificates": certs,
		"months":       months,
		"source":       "copernicus",
	})
}

func (h *ParcelHandler) persistTimeSeries(
	c *fiber.Ctx, cadastral string, parcel *entity.Parcel, ts *entity.SatelliteTimeSeries,
) []entity.NDVICertificate {
	if h.certRepo == nil || ts == nil || len(ts.Intervals) == 0 {
		return nil
	}

	certs := make([]entity.NDVICertificate, 0, len(ts.Intervals))
	for _, idx := range ts.Intervals {
		ndwi := idx.NDWI
		evi := idx.EVI
		lai := idx.LAI
		cf := idx.CloudFreePct
		sc := idx.SampleCount
		certs = append(certs, entity.NDVICertificate{
			ParcelID:        parcel.ID,
			CadastralNumber: cadastral,
			Season:          seasonFromTime(idx.PeriodStart),
			NDVIScore:       idx.NDVI,
			NDWIScore:       &ndwi,
			EVIScore:        &evi,
			LAIEstimate:     &lai,
			CloudFreePct:    &cf,
			SampleCount:     &sc,
			ObservedAt:      idx.PeriodStart,
		})
	}

	if err := h.certRepo.CreateBatch(c.Context(), certs); err != nil {
		h.logger.Warn().Err(err).Str("cadastral", cadastral).Msg("failed to persist satellite data")
		return nil
	}

	h.logger.Info().Str("cadastral", cadastral).Int("count", len(certs)).Msg("satellite data persisted")
	return certs
}

func seasonFromTime(t time.Time) string {
	q := (t.Month()-1)/3 + 1
	return fmt.Sprintf("%d-Q%d", t.Year(), q)
}

func (h *ParcelHandler) GetIndices(c *fiber.Ctx) error {
	cadastral := c.Params("cadastral")

	certs, err := h.certRepo.ListByParcel(c.Context(), cadastral)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch indices"})
	}

	ahi := ndvi.ComputeAHI(certs)

	return c.JSON(fiber.Map{
		"cadastral":    cadastral,
		"certificates": certs,
		"ahi":          ahi,
	})
}

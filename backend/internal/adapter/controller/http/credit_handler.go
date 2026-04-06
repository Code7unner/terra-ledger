package http

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/ndvi"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

const scoreMaxAge = 1 * time.Hour

type CreditHandler struct {
	parcelRepo  repository.ParcelRepo
	certRepo    repository.CertificateRepo
	lienRepo    repository.LienRepo
	scoreRepo   repository.CreditScoreRepo
	consentRepo repository.ConsentRepo
	scorer      repository.CreditScorer
	ndviUseCase *ndvi.UseCase
	logger      *zerolog.Logger
	inflight    sync.Map // cadastral → struct{}: dedup background goroutines
}

func NewCreditHandler(
	pr repository.ParcelRepo,
	cr repository.CertificateRepo,
	lr repository.LienRepo,
	sr repository.CreditScoreRepo,
	consentRepo repository.ConsentRepo,
	scorer repository.CreditScorer,
	ndviUseCase *ndvi.UseCase,
	logger *zerolog.Logger,
) *CreditHandler {
	return &CreditHandler{
		parcelRepo:  pr,
		certRepo:    cr,
		lienRepo:    lr,
		scoreRepo:   sr,
		consentRepo: consentRepo,
		scorer:      scorer,
		ndviUseCase: ndviUseCase,
		logger:      logger,
	}
}

func (h *CreditHandler) GetProfile(c *fiber.Ctx) error {
	cadastral := c.Params("cadastral")

	parcel, err := h.parcelRepo.GetByCadastral(c.Context(), cadastral)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "parcel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch parcel"})
	}

	// Check farmer consent — PDPA KZ Article 7 purpose limitation
	consentGranted := h.checkConsent(c, parcel.OwnerWallet)

	// Log lender access
	h.logAccess(c, parcel.OwnerWallet)

	if !consentGranted {
		return c.JSON(h.buildMaskedProfile(parcel))
	}

	return c.JSON(h.buildFullProfile(c, cadastral, parcel))
}

func (h *CreditHandler) checkConsent(c *fiber.Ctx, ownerWallet string) bool {
	if h.consentRepo == nil || ownerWallet == "" {
		return true // no consent repo or no owner = allow (graceful for demo)
	}

	consent, err := h.consentRepo.GetByWallet(c.Context(), ownerWallet)
	if err != nil {
		return true // consent not found = allow by default (farmer hasn't configured yet)
	}

	return consent.Status == entity.ConsentStatusGranted
}

func (h *CreditHandler) logAccess(c *fiber.Ctx, ownerWallet string) {
	if h.consentRepo == nil || ownerWallet == "" {
		return
	}

	lender, _ := c.Locals("lender").(*entity.Lender)
	if lender == nil {
		return
	}

	consent, err := h.consentRepo.GetByWallet(c.Context(), ownerWallet)
	if err != nil {
		return
	}

	_ = h.consentRepo.LogAccess(c.Context(), &entity.ConsentLogEntry{
		ConsentID:    consent.ID,
		LenderWallet: lender.ID.String(),
		LenderName:   lender.Name,
		DataType:     "credit_profile",
	})
}

func (h *CreditHandler) buildMaskedProfile(parcel *entity.Parcel) entity.CreditProfile {
	masked := *parcel
	masked.HolderName = "***"
	masked.HolderIINHash = "***"
	masked.OwnerWallet = masked.OwnerWallet[:8] + "..."

	return entity.CreditProfile{
		Parcel: masked,
		Productivity: entity.ProductivityData{
			NDVITrend:    "consent_required",
			DormancyRisk: "consent_required",
		},
		Encumbrances: entity.EncumbranceData{
			DoublePledgeRisk: false,
		},
	}
}

func (h *CreditHandler) buildFullProfile(
	c *fiber.Ctx, cadastral string, parcel *entity.Parcel,
) entity.CreditProfile {
	certs, _ := h.certRepo.ListByParcel(c.Context(), cadastral)

	// If no certs in DB, kick off background fetch + scoring and return partial profile.
	// Frontend retries and will get the full profile once background work completes.
	if len(certs) == 0 && h.ndviUseCase != nil && h.certRepo != nil {
		h.backgroundComputeScore(cadastral, *parcel)

		liens, _ := h.lienRepo.ListByParcel(c.Context(), cadastral)
		return entity.CreditProfile{
			Parcel: *parcel,
			Productivity: entity.ProductivityData{
				NDVITrend:    "computing",
				DormancyRisk: "unknown",
			},
			Encumbrances: entity.EncumbranceData{
				ActiveLiens:         filterActiveLiens(liens),
				LienCountHistorical: len(liens),
				DoublePledgeRisk:    len(filterActiveLiens(liens)) > 0,
			},
			// Credit is nil — frontend will retry
		}
	}

	liens, _ := h.lienRepo.ListByParcel(c.Context(), cadastral)

	activeLiens := filterActiveLiens(liens)
	score := h.resolveScore(c, cadastral, parcel, certs, activeLiens, liens)

	productivity := buildProductivityData(certs)

	return entity.CreditProfile{
		Parcel:       *parcel,
		Productivity: productivity,
		Encumbrances: entity.EncumbranceData{
			ActiveLiens:         activeLiens,
			LienCountHistorical: len(liens),
			DoublePledgeRisk:    len(activeLiens) > 0,
		},
		Credit: score,
	}
}

func buildProductivityData(certs []entity.NDVICertificate) entity.ProductivityData {
	pd := entity.ProductivityData{
		Certificates: certs,
		NDVITrend:    computeNDVITrend(certs),
		DormancyRisk: "low",
	}

	ndwiVals, eviVals := extractIndexValues(certs)
	if len(ndwiVals) > 0 {
		pd.NDWITrend = string(ndvi.ComputeIndexTrend(ndwiVals))
	}
	if len(eviVals) > 0 {
		pd.EVITrend = string(ndvi.ComputeIndexTrend(eviVals))
	}

	avgNDWI := avgFloat(ndwiVals)
	if ndvi.ComputeWaterStressRisk(avgNDWI) {
		pd.WaterStressRisk = "high"
	} else {
		pd.WaterStressRisk = "low"
	}

	return pd
}

func extractIndexValues(certs []entity.NDVICertificate) (ndwi, evi []float64) {
	for _, c := range certs {
		if c.NDWIScore != nil {
			ndwi = append(ndwi, *c.NDWIScore)
		}
		if c.EVIScore != nil {
			evi = append(evi, *c.EVIScore)
		}
	}
	return ndwi, evi
}

func avgFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func (h *CreditHandler) resolveScore(
	c *fiber.Ctx,
	cadastral string,
	parcel *entity.Parcel,
	certs []entity.NDVICertificate,
	activeLiens []entity.Encumbrance,
	allLiens []entity.Encumbrance,
) *entity.CreditScore {
	cached, _ := h.scoreRepo.GetByCadastral(c.Context(), cadastral)
	if cached != nil && time.Since(cached.ComputedAt) < scoreMaxAge {
		return cached
	}

	if h.scorer == nil {
		return cached
	}

	input := buildScoringInput(parcel, certs, activeLiens, allLiens)
	fresh, err := h.scorer.ComputeScore(c.Context(), input)
	if err != nil || fresh == nil {
		return cached
	}

	fresh.ParcelID = parcel.ID
	if err := h.scoreRepo.Upsert(c.Context(), fresh); err != nil {
		h.logger.Error().Err(err).Str("cadastral", cadastral).Msg("failed to cache credit score")
	}
	return fresh
}

// backgroundComputeScore fetches satellite data and computes AI score in a goroutine.
// Results are persisted to DB so the next /profile request returns them from cache.
// Uses sync.Map for dedup — concurrent retries for the same cadastral don't spawn extra goroutines.
func (h *CreditHandler) backgroundComputeScore(cadastral string, parcel entity.Parcel) {
	if _, loaded := h.inflight.LoadOrStore(cadastral, struct{}{}); loaded {
		return // already computing
	}

	go func() {
		defer h.inflight.Delete(cadastral)

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		certs, err := h.ndviUseCase.FetchAndStoreSeries(ctx, parcel, 12, h.certRepo)
		if err != nil {
			h.logger.Warn().Err(err).Str("cadastral", cadastral).Msg("background satellite fetch failed")
		}

		liens, _ := h.lienRepo.ListByParcel(ctx, cadastral)
		activeLiens := filterActiveLiens(liens)

		if h.scorer != nil {
			input := buildScoringInput(&parcel, certs, activeLiens, liens)
			score, err := h.scorer.ComputeScore(ctx, input)
			if err != nil {
				h.logger.Warn().Err(err).Str("cadastral", cadastral).Msg("background scoring failed")
				return
			}
			if score != nil {
				score.ParcelID = parcel.ID
				_ = h.scoreRepo.Upsert(ctx, score)
				h.logger.Info().Str("cadastral", cadastral).Int("score", score.AIScore).Msg("background score computed")
			}
		}
	}()
}

func buildScoringInput(
	parcel *entity.Parcel,
	certs []entity.NDVICertificate,
	activeLiens []entity.Encumbrance,
	allLiens []entity.Encumbrance,
) *entity.ScoringInput {
	input := &entity.ScoringInput{
		CadastralNumber: parcel.CadastralNumber,
		AreaHa:          parcel.AreaHa,
		LandClass:       parcel.LandClass,
		Oblast:          parcel.Oblast,
		NDVIHistory:     certs,
		ActiveLiens:     len(activeLiens),
		TotalLiens:      len(allLiens),
	}

	enrichInputFromCerts(input, certs)
	return input
}

func enrichInputFromCerts(input *entity.ScoringInput, certs []entity.NDVICertificate) {
	if len(certs) == 0 {
		return
	}

	ndviVals := make([]float64, 0, len(certs))
	var ndwiSum, eviSum float64
	var ndwiCount, eviCount int

	for _, c := range certs {
		ndviVals = append(ndviVals, c.NDVIScore)
		if c.NDWIScore != nil {
			ndwiSum += *c.NDWIScore
			ndwiCount++
		}
		if c.EVIScore != nil {
			eviSum += *c.EVIScore
			eviCount++
		}
	}

	input.NDVITrend = string(ndvi.ComputeIndexTrend(ndviVals))

	if ndwiCount > 0 {
		avgNDWI := ndwiSum / float64(ndwiCount)
		input.AvgNDWI = &avgNDWI
		input.WaterStressRisk = ndvi.ComputeWaterStressRisk(avgNDWI)
	}
	if eviCount > 0 {
		avgEVI := eviSum / float64(eviCount)
		input.AvgEVI = &avgEVI
	}
}

func filterActiveLiens(liens []entity.Encumbrance) []entity.Encumbrance {
	var active []entity.Encumbrance
	for _, l := range liens {
		if l.Status == entity.LienStatusActive {
			active = append(active, l)
		}
	}
	return active
}

func computeNDVITrend(certs []entity.NDVICertificate) string {
	if len(certs) < 2 {
		return "stable"
	}
	if certs[0].NDVIScore > certs[len(certs)-1].NDVIScore {
		return "improving"
	}
	if certs[0].NDVIScore < certs[len(certs)-1].NDVIScore {
		return "declining"
	}
	return "stable"
}

package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
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
}

func NewCreditHandler(
	pr repository.ParcelRepo,
	cr repository.CertificateRepo,
	lr repository.LienRepo,
	sr repository.CreditScoreRepo,
	consentRepo repository.ConsentRepo,
	scorer repository.CreditScorer,
) *CreditHandler {
	return &CreditHandler{
		parcelRepo:  pr,
		certRepo:    cr,
		lienRepo:    lr,
		scoreRepo:   sr,
		consentRepo: consentRepo,
		scorer:      scorer,
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
	liens, _ := h.lienRepo.ListByParcel(c.Context(), cadastral)

	activeLiens := filterActiveLiens(liens)
	score := h.resolveScore(c, cadastral, parcel, certs, activeLiens, liens)

	return entity.CreditProfile{
		Parcel: *parcel,
		Productivity: entity.ProductivityData{
			Certificates: certs,
			NDVITrend:    computeNDVITrend(certs),
			DormancyRisk: "low",
		},
		Encumbrances: entity.EncumbranceData{
			ActiveLiens:         activeLiens,
			LienCountHistorical: len(liens),
			DoublePledgeRisk:    len(activeLiens) > 0,
		},
		Credit: score,
	}
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
	_ = h.scoreRepo.Upsert(c.Context(), fresh)
	return fresh
}

func buildScoringInput(
	parcel *entity.Parcel,
	certs []entity.NDVICertificate,
	activeLiens []entity.Encumbrance,
	allLiens []entity.Encumbrance,
) *entity.ScoringInput {
	return &entity.ScoringInput{
		CadastralNumber: parcel.CadastralNumber,
		AreaHa:          parcel.AreaHa,
		LandClass:       parcel.LandClass,
		Oblast:          parcel.Oblast,
		NDVIHistory:     certs,
		ActiveLiens:     len(activeLiens),
		TotalLiens:      len(allLiens),
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

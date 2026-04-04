package http

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

var (
	discParcelRegistered      = eventDiscriminator("ParcelRegistered")
	discCertificateMinted     = eventDiscriminator("CertificateMinted")
	discEncumbranceRegistered = eventDiscriminator("EncumbranceRegistered")
	discEncumbranceReleased   = eventDiscriminator("EncumbranceReleased")
	discParcelDormant         = eventDiscriminator("ParcelDormant")
)

type HeliusTransaction struct {
	Signature    string              `json:"signature"`
	Timestamp    int64               `json:"timestamp"`
	Type         string              `json:"type"`
	AccountData  []HeliusAccountData `json:"accountData"`
	Instructions []HeliusInstruction `json:"instructions"`
	Meta         HeliusMeta          `json:"meta"`
}

type HeliusAccountData struct {
	Account      string `json:"account"`
	NativeChange int64  `json:"nativeBalanceChange"`
}

type HeliusInstruction struct {
	ProgramID string   `json:"programId"`
	Accounts  []string `json:"accounts"`
	Data      string   `json:"data"`
}

type HeliusMeta struct {
	LogMessages []string `json:"logMessages"`
}

type WebhookHandler struct {
	secret              string
	terraTokenProgramID string
	lienRegistryID      string
	parcelRepo          repository.ParcelRepo
	certRepo            repository.CertificateRepo
	lienRepo            repository.LienRepo
	logger              *zerolog.Logger
}

func NewWebhookHandler(
	secret, terraTokenID, lienRegistryID string,
	parcelRepo repository.ParcelRepo,
	certRepo repository.CertificateRepo,
	lienRepo repository.LienRepo,
	logger *zerolog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		secret:              secret,
		terraTokenProgramID: terraTokenID,
		lienRegistryID:      lienRegistryID,
		parcelRepo:          parcelRepo,
		certRepo:            certRepo,
		lienRepo:            lienRepo,
		logger:              logger,
	}
}

func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	if auth != h.secret {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	var txns []HeliusTransaction
	if err := c.BodyParser(&txns); err != nil {
		h.logger.Warn().Err(err).Msg("invalid webhook payload")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	for _, txn := range txns {
		h.processTransaction(c, txn)
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *WebhookHandler) processTransaction(c *fiber.Ctx, txn HeliusTransaction) {
	h.logger.Info().
		Str("signature", txn.Signature).
		Str("type", txn.Type).
		Msg("processing webhook transaction")

	for _, logLine := range txn.Meta.LogMessages {
		if !strings.HasPrefix(logLine, "Program data: ") {
			continue
		}

		data := strings.TrimPrefix(logLine, "Program data: ")
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil || len(decoded) < 8 {
			continue
		}

		var disc [8]byte
		copy(disc[:], decoded[:8])
		eventData := decoded[8:]

		h.routeEvent(c, txn, disc, eventData)
	}
}

func (h *WebhookHandler) routeEvent(c *fiber.Ctx, txn HeliusTransaction, disc [8]byte, data []byte) {
	switch disc {
	case discParcelRegistered:
		h.handleParcelRegistered(c, txn, data)
	case discCertificateMinted:
		h.handleCertificateMinted(c, txn, data)
	case discEncumbranceRegistered:
		h.handleEncumbranceRegistered(c, txn, data)
	case discEncumbranceReleased:
		h.handleEncumbranceReleased(c, txn, data)
	case discParcelDormant:
		h.handleParcelDormant(c, txn, data)
	}
}

// ParcelRegistered: { cadastral_number: String, owner: Pubkey, area_ha: u32 }
func (h *WebhookHandler) handleParcelRegistered(c *fiber.Ctx, txn HeliusTransaction, data []byte) {
	cadastral, offset, err := decodeBorshString(data, 0)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode ParcelRegistered cadastral")
		return
	}

	owner, _, err := decodePubkey(data, offset)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode ParcelRegistered owner")
		return
	}

	h.logger.Info().Str("sig", txn.Signature).Str("cadastral", cadastral).Str("owner", owner).Msg("event: ParcelRegistered")

	if err := h.parcelRepo.UpdateOnChainAddress(c.Context(), cadastral, owner); err != nil {
		h.logger.Warn().Err(err).Str("cadastral", cadastral).Msg("webhook: failed to update parcel on-chain address")
	}
}

// CertificateMinted: { cadastral_number: String, season: String, ndvi_score: u16, cert_address: Pubkey }
func (h *WebhookHandler) handleCertificateMinted(c *fiber.Ctx, txn HeliusTransaction, data []byte) {
	cadastral, offset, err := decodeBorshString(data, 0)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode CertificateMinted cadastral")
		return
	}

	season, offset, err := decodeBorshString(data, offset)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode CertificateMinted season")
		return
	}

	ndviScore, offset, err := decodeU16LE(data, offset)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode CertificateMinted ndvi_score")
		return
	}

	certAddr, _, err := decodePubkey(data, offset)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode CertificateMinted cert_address")
		return
	}

	h.logger.Info().
		Str("sig", txn.Signature).Str("cadastral", cadastral).
		Str("season", season).Uint16("ndvi", ndviScore).
		Msg("event: CertificateMinted")

	cert := &entity.NDVICertificate{
		CadastralNumber: cadastral,
		Season:          season,
		NDVIScore:       float64(ndviScore) / 1000.0,
		OnChainAddress:  certAddr,
		TxSignature:     txn.Signature,
	}
	if err := h.certRepo.Create(c.Context(), cert); err != nil {
		h.logger.Warn().Err(err).Msg("webhook: failed to create certificate")
	}
}

// EncumbranceRegistered: { cadastral_number: String, lender: Pubkey, amount: u64, parcel_pda: Pubkey }
func (h *WebhookHandler) handleEncumbranceRegistered(c *fiber.Ctx, txn HeliusTransaction, data []byte) {
	cadastral, offset, err := decodeBorshString(data, 0)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode EncumbranceRegistered cadastral")
		return
	}

	lender, offset, err := decodePubkey(data, offset)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode EncumbranceRegistered lender")
		return
	}

	amount, _, err := decodeU64LE(data, offset)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode EncumbranceRegistered amount")
		return
	}

	h.logger.Info().
		Str("sig", txn.Signature).Str("cadastral", cadastral).
		Str("lender", lender).Uint64("amount", amount).
		Msg("event: EncumbranceRegistered")

	lien := &entity.Encumbrance{
		CadastralNumber: cadastral,
		LenderWallet:    lender,
		AmountTenge:     int64(amount),
		TxSignature:     txn.Signature,
		Status:          entity.LienStatusActive,
	}
	if err := h.lienRepo.Create(c.Context(), lien); err != nil {
		h.logger.Warn().Err(err).Msg("webhook: failed to create lien")
	}
}

// EncumbranceReleased: { cadastral_number: String, lender: Pubkey, parcel_pda: Pubkey }
func (h *WebhookHandler) handleEncumbranceReleased(c *fiber.Ctx, txn HeliusTransaction, data []byte) {
	cadastral, offset, err := decodeBorshString(data, 0)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode EncumbranceReleased cadastral")
		return
	}

	lender, _, err := decodePubkey(data, offset)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode EncumbranceReleased lender")
		return
	}

	h.logger.Info().
		Str("sig", txn.Signature).Str("cadastral", cadastral).
		Str("lender", lender).Msg("event: EncumbranceReleased")

	existing, err := h.lienRepo.FindByWalletAndCadastral(c.Context(), lender, cadastral)
	if err != nil {
		h.logger.Warn().Err(err).Msg("webhook: failed to find lien for release")
		return
	}

	if err := h.lienRepo.UpdateStatus(c.Context(), existing.ID.String(), entity.LienStatusReleased); err != nil {
		h.logger.Warn().Err(err).Msg("webhook: failed to release lien")
	}
}

// ParcelDormant: { cadastral_number: String, seasons_dormant: u8 }
func (h *WebhookHandler) handleParcelDormant(c *fiber.Ctx, txn HeliusTransaction, data []byte) {
	cadastral, offset, err := decodeBorshString(data, 0)
	if err != nil {
		h.logger.Warn().Err(err).Str("sig", txn.Signature).Msg("failed to decode ParcelDormant cadastral")
		return
	}

	seasonsDormant := uint8(0)
	if offset < len(data) {
		seasonsDormant = data[offset]
	}

	h.logger.Warn().
		Str("sig", txn.Signature).
		Str("cadastral", cadastral).
		Uint8("seasons_dormant", seasonsDormant).
		Msg("event: ParcelDormant — parcel has no NDVI submissions")
	_ = c
}

func eventDiscriminator(name string) [8]byte {
	h := sha256.Sum256([]byte("event:" + name))
	var d [8]byte
	copy(d[:], h[:8])
	return d
}

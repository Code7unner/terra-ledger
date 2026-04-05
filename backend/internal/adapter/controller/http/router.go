package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type Handlers struct {
	Parcel      *ParcelHandler
	Lien        *LienHandler
	Credit      *CreditHandler
	Certificate *CertificateHandler
	Webhook     *WebhookHandler
	Consent     *ConsentHandler
	Logger      *zerolog.Logger
}

func RegisterRoutes(app *fiber.App, h *Handlers, lenderRepo repository.LenderRepo) {
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(RequestLogger(h.Logger))

	// Public
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/ready", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ready"})
	})

	// Webhook (own auth)
	app.Post("/webhooks/helius", h.Webhook.Handle)

	// Phantom mobile deeplink relay (no auth — public endpoints)
	phantom := NewPhantomHandler()
	app.Post("/api/v1/phantom/session", phantom.CreateSession)
	app.Get("/api/v1/phantom/callback", phantom.Callback)
	app.Get("/api/v1/phantom/poll/:session", phantom.Poll)

	// Authenticated API
	api := app.Group("/api/v1", APIKeyAuth(lenderRepo, h.Logger))

	// Parcels
	api.Post("/parcels", h.Parcel.Register)
	api.Get("/parcels/:cadastral", h.Parcel.Get)
	api.Get("/parcels/:cadastral/profile", h.Credit.GetProfile)
	api.Get("/parcels/:cadastral/ndvi", h.Parcel.GetNDVI)

	// Certificates
	api.Post("/parcels/:cadastral/certificates", h.Certificate.Mint)
	api.Get("/parcels/:cadastral/certificates", h.Certificate.List)

	// Liens
	api.Post("/liens", h.Lien.Register)
	api.Post("/liens/:id/release", h.Lien.Release)
	api.Get("/parcels/:cadastral/liens", h.Lien.ListByParcel)

	// Consent
	api.Post("/consent/grant", h.Consent.Grant)
	api.Post("/consent/revoke", h.Consent.Revoke)
	api.Get("/consent/:wallet", h.Consent.Get)
	api.Get("/consent/:wallet/log", h.Consent.ListAccessLog)
}

package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	handler "github.com/code7unner/decentrathon5/terra-ledger/backend/internal/adapter/controller/http"
	repo "github.com/code7unner/decentrathon5/terra-ledger/backend/internal/adapter/repository"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/infrastructure/config"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/infrastructure/migration"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/infrastructure/service"
)

func Start() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Config
	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("parsing config: %v\n", err)
		os.Exit(1)
	}

	// Logger
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger().Level(level)

	// Database
	db, err := service.NewPostgres(ctx, cfg.DatabaseURL, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Migrations
	if err := migration.Run(db, &logger); err != nil {
		logger.Fatal().Err(err).Msg("failed to run migrations")
	}

	// Repositories
	parcelRepo := repo.NewParcelPG(db, &logger)
	certRepo := repo.NewCertificatePG(db, &logger)
	lienRepo := repo.NewLienPG(db, &logger)
	lenderRepo := repo.NewLenderPG(db, &logger)
	scoreRepo := repo.NewCreditScorePG(db, &logger)

	// External adapters
	solanaRPC := repo.NewSolanaRPC(cfg.SolanaRPCURL, &logger)
	claudeScorer := repo.NewClaudeScorer(cfg.AnthropicAPIKey, cfg.AnthropicModel, &logger)
	// TODO: Phase 3
	_ = repo.NewCopernicusClient(cfg.CopernicusClientID, cfg.CopernicusClientSecret, &logger)

	// Signatures
	signatureRepo := repo.NewSignaturePG(db, &logger)

	// Consent
	consentRepo := repo.NewConsentPG(db, &logger)

	// Handlers
	parcelHandler := handler.NewParcelHandler(parcelRepo, solanaRPC, &logger)
	certHandler := handler.NewCertificateHandler(certRepo, parcelRepo)
	lienHandler := handler.NewLienHandler(lienRepo, parcelRepo, solanaRPC, &logger)
	creditHandler := handler.NewCreditHandler(parcelRepo, certRepo, lienRepo, scoreRepo, consentRepo, claudeScorer)
	consentHandler := handler.NewConsentHandler(consentRepo)
	webhookHandler := handler.NewWebhookHandler(
		cfg.HeliusWebhookSecret,
		cfg.TerraTokenProgramID,
		cfg.LienRegistryProgramID,
		parcelRepo, certRepo, lienRepo,
		&logger,
	)

	handlers := &handler.Handlers{
		Parcel:      parcelHandler,
		Lien:        lienHandler,
		Credit:      creditHandler,
		Certificate: certHandler,
		Webhook:     webhookHandler,
		Consent:     consentHandler,
		Logger:      &logger,
	}

	// Fiber app
	app := fiber.New(fiber.Config{
		AppName: cfg.AppName,
	})

	handler.RegisterRoutes(app, handlers, lenderRepo)

	// Load relay keypair for keeper
	relayKey, err := repo.LoadKeypair(cfg.RelayKeypairPath)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to load relay keypair, keeper will log only")
	}

	// Background services
	keeper := service.NewKeeper(solanaRPC, parcelRepo, cfg.KeeperInterval, relayKey, cfg.TerraTokenProgramID, &logger)
	go keeper.Start(ctx)

	reconciler := service.NewReconciler(
		solanaRPC, signatureRepo, cfg.TerraTokenProgramID, cfg.LienRegistryProgramID,
		60*time.Second, &logger,
	)
	go reconciler.Start(ctx)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info().Msg("shutting down...")
		cancel()
		_ = app.Shutdown()
	}()

	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Info().Str("addr", addr).Msg("starting server")
	if err := app.Listen(addr); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
	}
}

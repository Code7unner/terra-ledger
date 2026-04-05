package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type Reconciler struct {
	solana         repository.SolanaClient
	sigRepo        repository.SignatureRepo
	terraTokenID   string
	lienRegistryID string
	interval       time.Duration
	logger         *zerolog.Logger
}

func NewReconciler(
	solana repository.SolanaClient,
	sigRepo repository.SignatureRepo,
	terraTokenID, lienRegistryID string,
	interval time.Duration,
	logger *zerolog.Logger,
) *Reconciler {
	return &Reconciler{
		solana:         solana,
		sigRepo:        sigRepo,
		terraTokenID:   terraTokenID,
		lienRegistryID: lienRegistryID,
		interval:       interval,
		logger:         logger,
	}
}

func (r *Reconciler) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.logger.Info().Dur("interval", r.interval).Msg("reconciler started")

	for {
		select {
		case <-ctx.Done():
			r.logger.Info().Msg("reconciler stopped")
			return
		case <-ticker.C:
			r.processTick(ctx)
		}
	}
}

func (r *Reconciler) processTick(ctx context.Context) {
	r.reconcileProgram(ctx, "terra_token", r.terraTokenID)
	r.reconcileProgram(ctx, "lien_registry", r.lienRegistryID)
}

func (r *Reconciler) reconcileProgram(ctx context.Context, name, programID string) {
	sigs, err := r.solana.GetSignaturesForAddress(ctx, programID, 20)
	if err != nil {
		r.logger.Warn().Err(err).Str("program", name).Msg("reconciler fetch failed")
		return
	}

	r.logger.Info().
		Str("program", name).
		Int("signatures", len(sigs)).
		Msg("reconciler fetched signatures")

	for _, sig := range sigs {
		r.processSignature(ctx, sig, name, programID)
	}
}

func (r *Reconciler) processSignature(ctx context.Context, sig, name, programID string) {
	exists, err := r.sigRepo.SignatureExists(ctx, sig)
	if err != nil {
		r.logger.Error().Err(err).Str("sig", sig).Msg("check signature failed")
		return
	}

	if exists {
		return
	}

	r.logger.Info().
		Str("sig", sig).
		Str("program", name).
		Msg("detected unprocessed signature")

	if err := r.sigRepo.RecordSignature(ctx, sig, programID); err != nil {
		r.logger.Error().Err(err).Str("sig", sig).Msg("record signature failed")
	}
}

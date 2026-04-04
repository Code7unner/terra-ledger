package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type Reconciler struct {
	solana         repository.SolanaClient
	terraTokenID   string
	lienRegistryID string
	interval       time.Duration
	logger         *zerolog.Logger
}

func NewReconciler(
	solana repository.SolanaClient,
	terraTokenID, lienRegistryID string,
	interval time.Duration,
	logger *zerolog.Logger,
) *Reconciler {
	return &Reconciler{
		solana:         solana,
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

	// TODO: Compare against known tx_signatures in PG, process missing ones
}

package service

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type Keeper struct {
	solana          repository.SolanaClient
	parcels         repository.ParcelRepo
	interval        time.Duration
	relayKey        solana.PrivateKey
	terraTokenProgram string
	logger          *zerolog.Logger
}

func NewKeeper(
	solana repository.SolanaClient,
	parcels repository.ParcelRepo,
	interval time.Duration,
	relayKey solana.PrivateKey,
	terraTokenProgram string,
	logger *zerolog.Logger,
) *Keeper {
	return &Keeper{
		solana:            solana,
		parcels:           parcels,
		interval:          interval,
		relayKey:          relayKey,
		terraTokenProgram: terraTokenProgram,
		logger:            logger,
	}
}

func (k *Keeper) Start(ctx context.Context) {
	ticker := time.NewTicker(k.interval)
	defer ticker.Stop()

	k.logger.Info().Dur("interval", k.interval).Msg("keeper bot started")

	for {
		select {
		case <-ctx.Done():
			k.logger.Info().Msg("keeper bot stopped")
			return
		case <-ticker.C:
			k.processTick(ctx)
		}
	}
}

func (k *Keeper) processTick(ctx context.Context) {
	parcels, err := k.parcels.ListNeedingSeasonalCheck(ctx, k.interval)
	if err != nil {
		k.logger.Warn().Err(err).Msg("keeper: failed to list parcels")
		return
	}

	if len(parcels) == 0 {
		k.logger.Debug().Msg("keeper: no parcels need seasonal check")
		return
	}

	k.logger.Info().Int("count", len(parcels)).Msg("keeper: processing seasonal checks")

	for _, p := range parcels {
		if err := k.processParcel(ctx, p); err != nil {
			k.logger.Warn().
				Err(err).
				Str("cadastral", p.CadastralNumber).
				Msg("keeper: seasonal check failed")
		}
	}
}

func (k *Keeper) processParcel(ctx context.Context, p entity.Parcel) error {
	txData := buildSeasonalCheckData(p.CadastralNumber)

	k.logger.Info().
		Str("cadastral", p.CadastralNumber).
		Int("tx_bytes", len(txData)).
		Msg("keeper: built seasonal_check instruction data")

	// TODO: Build full transaction with relay keypair, send via SolanaRPC
	// For now, log the intent. Full tx building requires:
	// 1. Get recent blockhash
	// 2. Build transaction message with instruction
	// 3. Sign with relay keypair
	// 4. Serialize and send
	_ = ctx

	return nil
}

// buildSeasonalCheckData builds the Anchor instruction data for seasonal_check.
// Format: 8-byte discriminator + borsh-encoded cadastral string
func buildSeasonalCheckData(cadastral string) []byte {
	disc := instructionDiscriminator("seasonal_check")
	strLen := len(cadastral)
	data := make([]byte, 8+4+strLen)
	copy(data[:8], disc[:])
	binary.LittleEndian.PutUint32(data[8:12], uint32(strLen))
	copy(data[12:], cadastral)
	return data
}

func instructionDiscriminator(name string) [8]byte {
	h := sha256.Sum256([]byte("global:" + name))
	var d [8]byte
	copy(d[:], h[:8])
	return d
}

package service

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/ndvi"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type Keeper struct {
	solana            repository.SolanaClient
	parcels           repository.ParcelRepo
	certRepo          repository.CertificateRepo
	scoreRepo         repository.CreditScoreRepo
	decisionRepo      repository.AgentDecisionRepo
	scorer            repository.CreditScorer
	ndviUseCase       *ndvi.UseCase
	interval          time.Duration
	relayKey          solana.PrivateKey
	terraTokenProgram string
	logger            *zerolog.Logger
}

func NewKeeper(
	solana repository.SolanaClient,
	parcels repository.ParcelRepo,
	certRepo repository.CertificateRepo,
	scoreRepo repository.CreditScoreRepo,
	decisionRepo repository.AgentDecisionRepo,
	scorer repository.CreditScorer,
	ndviUseCase *ndvi.UseCase,
	interval time.Duration,
	relayKey solana.PrivateKey,
	terraTokenProgram string,
	logger *zerolog.Logger,
) *Keeper {
	return &Keeper{
		solana:            solana,
		parcels:           parcels,
		certRepo:          certRepo,
		scoreRepo:         scoreRepo,
		decisionRepo:      decisionRepo,
		scorer:            scorer,
		ndviUseCase:       ndviUseCase,
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
	// Fetch and store satellite time series if usecase available.
	var certs []entity.NDVICertificate
	if k.ndviUseCase != nil && k.certRepo != nil {
		var fetchErr error
		certs, fetchErr = k.ndviUseCase.FetchAndStoreSeries(ctx, p, 12, k.certRepo)
		if fetchErr != nil {
			k.logger.Warn().Err(fetchErr).
				Str("cadastral", p.CadastralNumber).
				Msg("keeper: satellite fetch failed, continuing")
		}
	}

	if err := k.sendSeasonalCheck(ctx, p); err != nil {
		return fmt.Errorf("seasonal check: %w", err)
	}

	prevScore, _ := k.scoreRepo.GetByCadastral(ctx, p.CadastralNumber)
	newScore := k.recalculateAndGetScore(ctx, p, certs)
	k.sendRiskAssessment(ctx, p, newScore, prevScore)

	return nil
}

func (k *Keeper) sendSeasonalCheck(ctx context.Context, p entity.Parcel) error {
	txData := buildSeasonalCheckData(p.CadastralNumber)

	k.logger.Info().
		Str("cadastral", p.CadastralNumber).
		Int("tx_bytes", len(txData)).
		Msg("keeper: built seasonal_check instruction data")

	txBytes, err := k.buildTransaction(ctx, p.CadastralNumber, txData)
	if err != nil {
		return fmt.Errorf("build transaction: %w", err)
	}

	sig, err := k.solana.SendTransaction(ctx, txBytes)
	if err != nil {
		return fmt.Errorf("send transaction: %w", err)
	}

	k.logger.Info().
		Str("cadastral", p.CadastralNumber).
		Str("signature", sig).
		Msg("keeper: seasonal_check transaction sent")

	return nil
}

func (k *Keeper) recalculateAndGetScore(ctx context.Context, p entity.Parcel, certs []entity.NDVICertificate) *entity.CreditScore {
	if k.scorer == nil || k.scoreRepo == nil {
		return nil
	}

	input := buildEnrichedScoringInput(p, certs)

	score, err := k.scorer.ComputeScore(ctx, input)
	if err != nil || score == nil {
		k.logger.Warn().Err(err).
			Str("cadastral", p.CadastralNumber).
			Msg("keeper: scoring failed")
		return nil
	}

	score.ParcelID = p.ID
	if err := k.scoreRepo.Upsert(ctx, score); err != nil {
		k.logger.Warn().Err(err).
			Str("cadastral", p.CadastralNumber).
			Msg("keeper: score upsert failed")
	}

	return score
}

func buildEnrichedScoringInput(p entity.Parcel, certs []entity.NDVICertificate) *entity.ScoringInput {
	input := &entity.ScoringInput{
		CadastralNumber: p.CadastralNumber,
		AreaHa:          p.AreaHa,
		LandClass:       p.LandClass,
		Oblast:          p.Oblast,
		NDVIHistory:     certs,
	}

	enrichScoringFromCerts(input, certs)
	return input
}

func enrichScoringFromCerts(input *entity.ScoringInput, certs []entity.NDVICertificate) {
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

func (k *Keeper) sendRiskAssessment(
	ctx context.Context,
	p entity.Parcel,
	score *entity.CreditScore,
	prevScore *entity.CreditScore,
) {
	if score == nil || k.relayKey == nil {
		return
	}

	if prevScore != nil && !isScoreChanged(score, prevScore) {
		return
	}

	grade := gradeToU8(score.CollateralGrade)
	ltv := uint16(score.RecommendedLTV * 10000)
	txData := buildRiskAssessmentData(p.CadastralNumber, uint8(score.AIScore), grade, ltv)

	txBytes, err := k.buildRiskAssessmentTx(ctx, p.CadastralNumber, txData)
	if err != nil {
		k.logger.Warn().Err(err).
			Str("cadastral", p.CadastralNumber).
			Msg("keeper: failed to build risk assessment tx")
		return
	}

	sig, err := k.solana.SendTransaction(ctx, txBytes)
	if err != nil {
		k.logger.Warn().Err(err).
			Str("cadastral", p.CadastralNumber).
			Msg("keeper: failed to send risk assessment tx")
		return
	}

	k.logger.Info().
		Str("cadastral", p.CadastralNumber).
		Str("sig", sig).Int("score", score.AIScore).
		Msg("keeper: risk assessment sent on-chain")

	k.recordDecision(ctx, p, score, prevScore, sig)
}

func isScoreChanged(score, prevScore *entity.CreditScore) bool {
	delta := score.AIScore - prevScore.AIScore
	if delta < 0 {
		delta = -delta
	}

	return delta >= 5 || score.CollateralGrade != prevScore.CollateralGrade
}

func (k *Keeper) recordDecision(
	ctx context.Context,
	p entity.Parcel,
	score *entity.CreditScore,
	prevScore *entity.CreditScore,
	sig string,
) {
	if k.decisionRepo == nil {
		return
	}

	decision := &entity.AgentDecision{
		ParcelID:        p.ID,
		CadastralNumber: p.CadastralNumber,
		NewScore:        score.AIScore,
		NewGrade:        score.CollateralGrade,
		Reason:          score.Explanation,
		TxSignature:     sig,
	}

	if prevScore != nil {
		decision.PreviousScore = &prevScore.AIScore
		decision.PreviousGrade = &prevScore.CollateralGrade
	}

	if err := k.decisionRepo.Create(ctx, decision); err != nil {
		k.logger.Warn().Err(err).Msg("keeper: failed to record agent decision")
	}
}

func gradeToU8(grade string) uint8 {
	switch grade {
	case "A":
		return 0
	case "B":
		return 1
	case "C":
		return 2
	default:
		return 3
	}
}

func buildRiskAssessmentData(cadastral string, aiScore, grade uint8, ltv uint16) []byte {
	disc := instructionDiscriminator("update_risk_assessment")
	strLen := len(cadastral)
	data := make([]byte, 8+4+strLen+1+1+2)
	copy(data[:8], disc[:])
	binary.LittleEndian.PutUint32(data[8:12], uint32(strLen))
	copy(data[12:12+strLen], cadastral)
	off := 12 + strLen
	data[off] = aiScore
	data[off+1] = grade
	binary.LittleEndian.PutUint16(data[off+2:off+4], ltv)
	return data
}

func (k *Keeper) buildRiskAssessmentTx(
	ctx context.Context,
	cadastral string,
	ixData []byte,
) ([]byte, error) {
	blockhash, err := k.solana.GetRecentBlockhash(ctx)
	if err != nil {
		return nil, fmt.Errorf("get recent blockhash: %w", err)
	}

	programID := solana.MustPublicKeyFromBase58(k.terraTokenProgram)
	parcelPDA := deriveParcelPDA(programID, cadastral)
	relayPub := k.relayKey.PublicKey()

	ix := buildRiskAssessmentInstruction(programID, parcelPDA, relayPub, ixData)

	bh := solana.MustHashFromBase58(blockhash)
	tx, err := solana.NewTransaction(
		[]solana.Instruction{ix},
		bh,
		solana.TransactionPayer(relayPub),
	)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	if _, err := tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(relayPub) {
			return &k.relayKey
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	raw, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("serialize transaction: %w", err)
	}

	return raw, nil
}

func buildRiskAssessmentInstruction(
	programID, parcelPDA, authority solana.PublicKey,
	data []byte,
) *seasonalCheckIx {
	return &seasonalCheckIx{
		programID: programID,
		accounts: solana.AccountMetaSlice{
			solana.NewAccountMeta(parcelPDA, true, false),
			solana.NewAccountMeta(authority, true, true),
		},
		data: data,
	}
}

func (k *Keeper) buildTransaction(
	ctx context.Context,
	cadastral string,
	ixData []byte,
) ([]byte, error) {
	blockhash, err := k.solana.GetRecentBlockhash(ctx)
	if err != nil {
		return nil, fmt.Errorf("get recent blockhash: %w", err)
	}

	programID := solana.MustPublicKeyFromBase58(k.terraTokenProgram)
	parcelPDA := deriveParcelPDA(programID, cadastral)
	relayPub := k.relayKey.PublicKey()

	ix := buildSeasonalCheckInstruction(programID, parcelPDA, relayPub, ixData)

	bh := solana.MustHashFromBase58(blockhash)
	tx, err := solana.NewTransaction(
		[]solana.Instruction{ix},
		bh,
		solana.TransactionPayer(relayPub),
	)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	if _, err := tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(relayPub) {
			return &k.relayKey
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	raw, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("serialize transaction: %w", err)
	}

	return raw, nil
}

func deriveParcelPDA(programID solana.PublicKey, cadastral string) solana.PublicKey {
	addr, _, _ := solana.FindProgramAddress(
		[][]byte{[]byte("parcel"), []byte(cadastral)},
		programID,
	)
	return addr
}

func buildSeasonalCheckInstruction(
	programID, parcelPDA, authority solana.PublicKey,
	data []byte,
) *seasonalCheckIx {
	return &seasonalCheckIx{
		programID: programID,
		accounts: solana.AccountMetaSlice{
			solana.NewAccountMeta(parcelPDA, true, false),
			solana.NewAccountMeta(authority, true, true),
		},
		data: data,
	}
}

// seasonalCheckIx implements solana.Instruction for the seasonal_check call.
type seasonalCheckIx struct {
	programID solana.PublicKey
	accounts  solana.AccountMetaSlice
	data      []byte
}

func (ix *seasonalCheckIx) ProgramID() solana.PublicKey     { return ix.programID }
func (ix *seasonalCheckIx) Accounts() []*solana.AccountMeta { return ix.accounts }
func (ix *seasonalCheckIx) Data() ([]byte, error)           { return ix.data, nil }

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

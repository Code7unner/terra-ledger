package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

type CreditScorePG struct {
	db     *sql.DB
	logger *zerolog.Logger
}

func NewCreditScorePG(db *sql.DB, logger *zerolog.Logger) *CreditScorePG {
	return &CreditScorePG{db: db, logger: logger}
}

func (r *CreditScorePG) Upsert(ctx context.Context, score *entity.CreditScore) error {
	riskJSON, err := json.Marshal(score.RiskFactors)
	if err != nil {
		return fmt.Errorf("marshalling risk factors: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `
		INSERT INTO credit_scores (
			parcel_id, cadastral_number, ai_score, recommended_ltv,
			collateral_grade, estimated_value_tenge, model_version,
			explanation, risk_factors
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (cadastral_number) DO UPDATE SET
			ai_score = EXCLUDED.ai_score,
			recommended_ltv = EXCLUDED.recommended_ltv,
			collateral_grade = EXCLUDED.collateral_grade,
			estimated_value_tenge = EXCLUDED.estimated_value_tenge,
			model_version = EXCLUDED.model_version,
			explanation = EXCLUDED.explanation,
			risk_factors = EXCLUDED.risk_factors,
			computed_at = NOW()
		RETURNING id, computed_at`,
		score.ParcelID, score.CadastralNumber, score.AIScore, score.RecommendedLTV,
		score.CollateralGrade, score.EstimatedValueTenge, score.ModelVersion,
		score.Explanation, riskJSON,
	).Scan(&score.ID, &score.ComputedAt)
	if err != nil {
		return fmt.Errorf("upserting credit score: %w", err)
	}

	return nil
}

func (r *CreditScorePG) GetByCadastral(ctx context.Context, cadastral string) (*entity.CreditScore, error) {
	var (
		s        entity.CreditScore
		riskRaw  []byte
		grade    sql.NullString
		model    sql.NullString
		explain  sql.NullString
		aiScore  sql.NullInt32
		ltv      sql.NullFloat64
		estValue sql.NullInt64
	)

	err := r.db.QueryRowContext(ctx, `
		SELECT id, parcel_id, cadastral_number, ai_score, recommended_ltv,
		       collateral_grade, estimated_value_tenge, model_version,
		       explanation, risk_factors, computed_at
		FROM credit_scores
		WHERE cadastral_number = $1`, cadastral,
	).Scan(
		&s.ID, &s.ParcelID, &s.CadastralNumber, &aiScore, &ltv,
		&grade, &estValue, &model,
		&explain, &riskRaw, &s.ComputedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("querying credit score for %q: %w", cadastral, err)
	}

	s.AIScore = int(aiScore.Int32)
	s.RecommendedLTV = ltv.Float64
	s.CollateralGrade = grade.String
	s.EstimatedValueTenge = estValue.Int64
	s.ModelVersion = model.String
	s.Explanation = explain.String

	if len(riskRaw) > 0 {
		_ = json.Unmarshal(riskRaw, &s.RiskFactors)
	}

	return &s, nil
}

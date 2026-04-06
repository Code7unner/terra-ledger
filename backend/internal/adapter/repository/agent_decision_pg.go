package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

type AgentDecisionPG struct {
	db     *sql.DB
	logger *zerolog.Logger
}

func NewAgentDecisionPG(db *sql.DB, logger *zerolog.Logger) *AgentDecisionPG {
	return &AgentDecisionPG{db: db, logger: logger}
}

func (r *AgentDecisionPG) Create(ctx context.Context, d *entity.AgentDecision) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO agent_decisions
			(parcel_id, cadastral_number, previous_score, new_score,
			 previous_grade, new_grade, reason, tx_signature)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, decided_at`,
		d.ParcelID, d.CadastralNumber, d.PreviousScore, d.NewScore,
		d.PreviousGrade, d.NewGrade, d.Reason, d.TxSignature,
	).Scan(&d.ID, &d.DecidedAt)
	if err != nil {
		return fmt.Errorf("inserting agent decision: %w", err)
	}

	return nil
}

func (r *AgentDecisionPG) ListRecent(ctx context.Context, limit int) ([]entity.AgentDecision, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, parcel_id, cadastral_number, previous_score, new_score,
		       previous_grade, new_grade, reason, tx_signature, decided_at
		FROM agent_decisions
		ORDER BY decided_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("listing recent agent decisions: %w", err)
	}
	defer rows.Close()

	return scanAgentDecisions(rows)
}

func (r *AgentDecisionPG) ListByParcel(
	ctx context.Context,
	cadastral string,
	limit int,
) ([]entity.AgentDecision, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, parcel_id, cadastral_number, previous_score, new_score,
		       previous_grade, new_grade, reason, tx_signature, decided_at
		FROM agent_decisions
		WHERE cadastral_number = $1
		ORDER BY decided_at DESC
		LIMIT $2`, cadastral, limit)
	if err != nil {
		return nil, fmt.Errorf("listing agent decisions by parcel: %w", err)
	}
	defer rows.Close()

	return scanAgentDecisions(rows)
}

func scanAgentDecisions(rows *sql.Rows) ([]entity.AgentDecision, error) {
	var decisions []entity.AgentDecision

	for rows.Next() {
		var d entity.AgentDecision
		var (
			prevScore sql.NullInt32
			prevGrade sql.NullString
			txSig     sql.NullString
			parcelID  uuid.NullUUID
		)

		err := rows.Scan(
			&d.ID, &parcelID, &d.CadastralNumber,
			&prevScore, &d.NewScore,
			&prevGrade, &d.NewGrade,
			&d.Reason, &txSig, &d.DecidedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning agent decision: %w", err)
		}

		if parcelID.Valid {
			d.ParcelID = parcelID.UUID
		}
		if prevScore.Valid {
			v := int(prevScore.Int32)
			d.PreviousScore = &v
		}
		if prevGrade.Valid {
			d.PreviousGrade = &prevGrade.String
		}
		if txSig.Valid {
			d.TxSignature = txSig.String
		}

		decisions = append(decisions, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating agent decisions: %w", err)
	}

	return decisions, nil
}

package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/zerolog"
)

type SignaturePG struct {
	db     *sql.DB
	logger *zerolog.Logger
}

func NewSignaturePG(db *sql.DB, logger *zerolog.Logger) *SignaturePG {
	return &SignaturePG{db: db, logger: logger}
}

func (r *SignaturePG) SignatureExists(ctx context.Context, signature string) (bool, error) {
	var exists bool

	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM processed_signatures WHERE signature = $1)",
		signature,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check signature: %w", err)
	}

	return exists, nil
}

func (r *SignaturePG) RecordSignature(ctx context.Context, signature, programID string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO processed_signatures (signature, program_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		signature, programID,
	)
	if err != nil {
		return fmt.Errorf("record signature: %w", err)
	}

	return nil
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

type LenderPG struct {
	db     *sql.DB
	logger *zerolog.Logger
}

func NewLenderPG(db *sql.DB, logger *zerolog.Logger) *LenderPG {
	return &LenderPG{db: db, logger: logger}
}

func (r *LenderPG) GetByAPIKey(ctx context.Context, key string) (*entity.Lender, error) {
	var (
		l   entity.Lender
		bin sql.NullString
	)

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, bin, api_key, tier,
		       queries_this_month, query_limit, created_at
		FROM lenders
		WHERE api_key = $1`, key,
	).Scan(
		&l.ID, &l.Name, &bin, &l.APIKey, &l.Tier,
		&l.QueriesThisMonth, &l.QueryLimit, &l.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("querying lender by API key: %w", err)
	}

	l.BIN = bin.String

	return &l, nil
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

type ConsentPG struct {
	db     *sql.DB
	logger *zerolog.Logger
}

func NewConsentPG(db *sql.DB, logger *zerolog.Logger) *ConsentPG {
	return &ConsentPG{db: db, logger: logger}
}

func (r *ConsentPG) Grant(ctx context.Context, walletAddress string) (*entity.Consent, error) {
	now := time.Now()
	var c entity.Consent

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO consents (wallet_address, status, granted_at)
		VALUES ($1, 'granted', $2)
		ON CONFLICT (wallet_address) DO UPDATE
		SET status = 'granted', granted_at = $2, revoked_at = NULL
		RETURNING id, wallet_address, status, granted_at, revoked_at, created_at`,
		walletAddress, now,
	).Scan(&c.ID, &c.WalletAddress, &c.Status, &c.GrantedAt, &c.RevokedAt, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("granting consent: %w", err)
	}

	return &c, nil
}

func (r *ConsentPG) Revoke(ctx context.Context, walletAddress string) (*entity.Consent, error) {
	now := time.Now()
	var c entity.Consent

	err := r.db.QueryRowContext(ctx, `
		UPDATE consents SET status = 'revoked', revoked_at = $1
		WHERE wallet_address = $2
		RETURNING id, wallet_address, status, granted_at, revoked_at, created_at`,
		now, walletAddress,
	).Scan(&c.ID, &c.WalletAddress, &c.Status, &c.GrantedAt, &c.RevokedAt, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("revoking consent: %w", err)
	}

	return &c, nil
}

func (r *ConsentPG) GetByWallet(ctx context.Context, walletAddress string) (*entity.Consent, error) {
	var c entity.Consent

	err := r.db.QueryRowContext(ctx, `
		SELECT id, wallet_address, status, granted_at, revoked_at, created_at
		FROM consents WHERE wallet_address = $1`, walletAddress,
	).Scan(&c.ID, &c.WalletAddress, &c.Status, &c.GrantedAt, &c.RevokedAt, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("querying consent: %w", err)
	}

	return &c, nil
}

func (r *ConsentPG) LogAccess(ctx context.Context, entry *entity.ConsentLogEntry) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO consent_access_log (consent_id, lender_wallet, lender_name, data_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, accessed_at`,
		entry.ConsentID, entry.LenderWallet, entry.LenderName, entry.DataType,
	).Scan(&entry.ID, &entry.AccessedAt)
	if err != nil {
		return fmt.Errorf("logging consent access: %w", err)
	}
	return nil
}

func (r *ConsentPG) ListAccessLog(ctx context.Context, walletAddress string) ([]entity.ConsentLogEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT l.id, l.consent_id, l.lender_wallet, l.lender_name, l.data_type, l.accessed_at
		FROM consent_access_log l
		JOIN consents c ON c.id = l.consent_id
		WHERE c.wallet_address = $1
		ORDER BY l.accessed_at DESC
		LIMIT 100`, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("listing access log: %w", err)
	}
	defer rows.Close()

	var entries []entity.ConsentLogEntry
	for rows.Next() {
		var e entity.ConsentLogEntry
		var lenderName sql.NullString
		if err := rows.Scan(&e.ID, &e.ConsentID, &e.LenderWallet, &lenderName, &e.DataType, &e.AccessedAt); err != nil {
			return nil, fmt.Errorf("scanning access log entry: %w", err)
		}
		e.LenderName = lenderName.String
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

type LienPG struct {
	db     *sql.DB
	logger *zerolog.Logger
}

func NewLienPG(db *sql.DB, logger *zerolog.Logger) *LienPG {
	return &LienPG{db: db, logger: logger}
}

func (r *LienPG) Create(ctx context.Context, lien *entity.Encumbrance) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO liens (
			parcel_id, cadastral_number, lender_wallet, lender_name,
			amount_tenge, notary_cert_hash, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, registered_at`,
		lien.ParcelID, lien.CadastralNumber, lien.LenderWallet, lien.LenderName,
		lien.AmountTenge, lien.NotaryCertHash, lien.Status,
	).Scan(&lien.ID, &lien.RegisteredAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return entity.ErrDoublePledge
		}
		return fmt.Errorf("inserting lien: %w", err)
	}

	return nil
}

func (r *LienPG) GetByID(ctx context.Context, id string) (*entity.Encumbrance, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, parcel_id, cadastral_number, lender_wallet, lender_name,
		       amount_tenge, notary_cert_hash, on_chain_address, tx_signature,
		       status, registered_at, released_at
		FROM liens
		WHERE id = $1`, id)

	l, err := scanLienRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("querying lien by id %q: %w", id, err)
	}

	return l, nil
}

func (r *LienPG) GetActive(ctx context.Context, cadastral string) (*entity.Encumbrance, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, parcel_id, cadastral_number, lender_wallet, lender_name,
		       amount_tenge, notary_cert_hash, on_chain_address, tx_signature,
		       status, registered_at, released_at
		FROM liens
		WHERE cadastral_number = $1 AND status = 'active'
		LIMIT 1`, cadastral)

	l, err := scanLienRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("querying active lien for %q: %w", cadastral, err)
	}

	return l, nil
}

func (r *LienPG) ListByParcel(ctx context.Context, cadastral string) ([]entity.Encumbrance, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, parcel_id, cadastral_number, lender_wallet, lender_name,
		       amount_tenge, notary_cert_hash, on_chain_address, tx_signature,
		       status, registered_at, released_at
		FROM liens
		WHERE cadastral_number = $1
		ORDER BY registered_at DESC`, cadastral)
	if err != nil {
		return nil, fmt.Errorf("listing liens: %w", err)
	}
	defer rows.Close()

	var liens []entity.Encumbrance
	for rows.Next() {
		l, err := scanLienFromScanner(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning lien row: %w", err)
		}
		liens = append(liens, *l)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating lien rows: %w", err)
	}

	return liens, nil
}

func (r *LienPG) UpdateStatus(ctx context.Context, id string, status entity.LienStatus) error {
	query := `UPDATE liens SET status = $1 WHERE id = $2`
	if status == entity.LienStatusReleased {
		query = `UPDATE liens SET status = $1, released_at = NOW() WHERE id = $2`
	}

	res, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("updating lien status: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return entity.ErrNotFound
	}

	return nil
}

type lienScanner interface {
	Scan(dest ...any) error
}

func scanLienFromScanner(s lienScanner) (*entity.Encumbrance, error) {
	var (
		l        entity.Encumbrance
		name     sql.NullString
		notary   sql.NullString
		onChain  sql.NullString
		txSig    sql.NullString
	)

	err := s.Scan(
		&l.ID, &l.ParcelID, &l.CadastralNumber, &l.LenderWallet, &name,
		&l.AmountTenge, &notary, &onChain, &txSig,
		&l.Status, &l.RegisteredAt, &l.ReleasedAt,
	)
	if err != nil {
		return nil, err
	}

	l.LenderName = name.String
	l.NotaryCertHash = notary.String
	l.OnChainAddress = onChain.String
	l.TxSignature = txSig.String

	return &l, nil
}

func (r *LienPG) FindByWalletAndCadastral(ctx context.Context, lenderWallet, cadastral string) (*entity.Encumbrance, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, parcel_id, cadastral_number, lender_wallet, lender_name,
		       amount_tenge, notary_cert_hash, on_chain_address, tx_signature,
		       status, registered_at, released_at
		FROM liens
		WHERE lender_wallet = $1 AND cadastral_number = $2 AND status = 'active'
		LIMIT 1`, lenderWallet, cadastral)

	l, err := scanLienRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("querying lien by wallet+cadastral: %w", err)
	}

	return l, nil
}

func scanLienRow(row *sql.Row) (*entity.Encumbrance, error) {
	return scanLienFromScanner(row)
}

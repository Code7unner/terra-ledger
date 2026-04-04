package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

type CertificatePG struct {
	db     *sql.DB
	logger *zerolog.Logger
}

func NewCertificatePG(db *sql.DB, logger *zerolog.Logger) *CertificatePG {
	return &CertificatePG{db: db, logger: logger}
}

func (r *CertificatePG) Create(ctx context.Context, cert *entity.NDVICertificate) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO certificates (
			parcel_id, cadastral_number, season, ndvi_score,
			crop_type, yield_t_ha, sentinel_scene_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, minted_at`,
		cert.ParcelID, cert.CadastralNumber, cert.Season, cert.NDVIScore,
		cert.CropType, cert.YieldTHa, cert.SentinelSceneID,
	).Scan(&cert.ID, &cert.MintedAt)
	if err != nil {
		return fmt.Errorf("inserting certificate: %w", err)
	}

	return nil
}

func (r *CertificatePG) ListByParcel(ctx context.Context, cadastral string) ([]entity.NDVICertificate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, parcel_id, cadastral_number, season, ndvi_score,
		       crop_type, yield_t_ha, sentinel_scene_id,
		       on_chain_address, tx_signature, minted_at
		FROM certificates
		WHERE cadastral_number = $1
		ORDER BY minted_at DESC`, cadastral)
	if err != nil {
		return nil, fmt.Errorf("listing certificates: %w", err)
	}
	defer rows.Close()

	var certs []entity.NDVICertificate
	for rows.Next() {
		c, err := scanCertificate(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning certificate row: %w", err)
		}
		certs = append(certs, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating certificate rows: %w", err)
	}

	return certs, nil
}

func (r *CertificatePG) GetLatest(ctx context.Context, cadastral string) (*entity.NDVICertificate, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, parcel_id, cadastral_number, season, ndvi_score,
		       crop_type, yield_t_ha, sentinel_scene_id,
		       on_chain_address, tx_signature, minted_at
		FROM certificates
		WHERE cadastral_number = $1
		ORDER BY minted_at DESC
		LIMIT 1`, cadastral)

	c, err := scanCertificateRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("querying latest certificate for %q: %w", cadastral, err)
	}

	return c, nil
}

type certScanner interface {
	Scan(dest ...any) error
}

func scanCertificateFromScanner(s certScanner) (*entity.NDVICertificate, error) {
	var (
		c        entity.NDVICertificate
		crop     sql.NullString
		yield    sql.NullFloat64
		scene    sql.NullString
		onChain  sql.NullString
		txSig    sql.NullString
	)

	err := s.Scan(
		&c.ID, &c.ParcelID, &c.CadastralNumber, &c.Season, &c.NDVIScore,
		&crop, &yield, &scene, &onChain, &txSig, &c.MintedAt,
	)
	if err != nil {
		return nil, err
	}

	c.CropType = crop.String
	c.YieldTHa = yield.Float64
	c.SentinelSceneID = scene.String
	c.OnChainAddress = onChain.String
	c.TxSignature = txSig.String

	return &c, nil
}

func scanCertificate(rows *sql.Rows) (*entity.NDVICertificate, error) {
	return scanCertificateFromScanner(rows)
}

func scanCertificateRow(row *sql.Row) (*entity.NDVICertificate, error) {
	return scanCertificateFromScanner(row)
}

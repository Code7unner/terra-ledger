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
			ndwi_score, evi_score, lai_estimate, cloud_free_pct,
			sample_count, observed_at,
			crop_type, yield_t_ha, sentinel_scene_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, minted_at`,
		cert.ParcelID, cert.CadastralNumber, cert.Season, cert.NDVIScore,
		cert.NDWIScore, cert.EVIScore, cert.LAIEstimate, cert.CloudFreePct,
		cert.SampleCount, nullTimeVal(cert.ObservedAt),
		cert.CropType, cert.YieldTHa, cert.SentinelSceneID,
	).Scan(&cert.ID, &cert.MintedAt)
	if err != nil {
		return fmt.Errorf("inserting certificate: %w", err)
	}

	return nil
}

func (r *CertificatePG) CreateBatch(ctx context.Context, certs []entity.NDVICertificate) error {
	if len(certs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin batch tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for i := range certs {
		if err := r.insertCertInTx(ctx, tx, &certs[i]); err != nil {
			return fmt.Errorf("batch insert cert %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit batch tx: %w", err)
	}
	return nil
}

func (r *CertificatePG) insertCertInTx(
	ctx context.Context, tx *sql.Tx, cert *entity.NDVICertificate,
) error {
	return tx.QueryRowContext(ctx, `
		INSERT INTO certificates (
			parcel_id, cadastral_number, season, ndvi_score,
			ndwi_score, evi_score, lai_estimate, cloud_free_pct,
			sample_count, observed_at,
			crop_type, yield_t_ha, sentinel_scene_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (cadastral_number, observed_at)
		DO UPDATE SET
			ndvi_score    = EXCLUDED.ndvi_score,
			ndwi_score    = EXCLUDED.ndwi_score,
			evi_score     = EXCLUDED.evi_score,
			lai_estimate  = EXCLUDED.lai_estimate,
			cloud_free_pct = EXCLUDED.cloud_free_pct,
			sample_count  = EXCLUDED.sample_count
		RETURNING id, minted_at`,
		cert.ParcelID, cert.CadastralNumber, cert.Season, cert.NDVIScore,
		cert.NDWIScore, cert.EVIScore, cert.LAIEstimate, cert.CloudFreePct,
		cert.SampleCount, nullTimeVal(cert.ObservedAt),
		cert.CropType, cert.YieldTHa, cert.SentinelSceneID,
	).Scan(&cert.ID, &cert.MintedAt)
}

func (r *CertificatePG) ListByParcelInRange(
	ctx context.Context, cadastral string, from, to time.Time,
) ([]entity.NDVICertificate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, parcel_id, cadastral_number, season, ndvi_score,
		       ndwi_score, evi_score, lai_estimate, cloud_free_pct,
		       sample_count, observed_at,
		       crop_type, yield_t_ha, sentinel_scene_id,
		       on_chain_address, tx_signature, minted_at
		FROM certificates
		WHERE cadastral_number = $1
		  AND observed_at >= $2
		  AND observed_at <= $3
		ORDER BY observed_at ASC`, cadastral, from, to)
	if err != nil {
		return nil, fmt.Errorf("listing certificates in range: %w", err)
	}
	defer rows.Close()

	var certs []entity.NDVICertificate
	for rows.Next() {
		c, scanErr := scanCertificateExt(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning certificate row: %w", scanErr)
		}
		certs = append(certs, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating certificate rows: %w", err)
	}
	return certs, nil
}

func nullTimeVal(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (r *CertificatePG) ListByParcel(ctx context.Context, cadastral string) ([]entity.NDVICertificate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, parcel_id, cadastral_number, season, ndvi_score,
		       ndwi_score, evi_score, lai_estimate, cloud_free_pct,
		       sample_count, observed_at,
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
		c, scanErr := scanCertificateExt(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning certificate row: %w", scanErr)
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

// scanCertificateExt scans the extended column set including new satellite indices.
func scanCertificateExt(s certScanner) (*entity.NDVICertificate, error) {
	var (
		c           entity.NDVICertificate
		ndwi        sql.NullFloat64
		evi         sql.NullFloat64
		lai         sql.NullFloat64
		cloudFree   sql.NullFloat64
		sampleCount sql.NullInt32
		observedAt  sql.NullTime
		crop        sql.NullString
		yield       sql.NullFloat64
		scene       sql.NullString
		onChain     sql.NullString
		txSig       sql.NullString
	)

	err := s.Scan(
		&c.ID, &c.ParcelID, &c.CadastralNumber, &c.Season, &c.NDVIScore,
		&ndwi, &evi, &lai, &cloudFree,
		&sampleCount, &observedAt,
		&crop, &yield, &scene,
		&onChain, &txSig, &c.MintedAt,
	)
	if err != nil {
		return nil, err
	}

	assignExtFields(&c, ndwi, evi, lai, cloudFree, sampleCount, observedAt)
	c.CropType = crop.String
	c.YieldTHa = yield.Float64
	c.SentinelSceneID = scene.String
	c.OnChainAddress = onChain.String
	c.TxSignature = txSig.String

	return &c, nil
}

func assignExtFields(
	c *entity.NDVICertificate,
	ndwi, evi, lai, cloudFree sql.NullFloat64,
	sampleCount sql.NullInt32,
	observedAt sql.NullTime,
) {
	if ndwi.Valid {
		c.NDWIScore = &ndwi.Float64
	}
	if evi.Valid {
		c.EVIScore = &evi.Float64
	}
	if lai.Valid {
		c.LAIEstimate = &lai.Float64
	}
	if cloudFree.Valid {
		c.CloudFreePct = &cloudFree.Float64
	}
	if sampleCount.Valid {
		v := int(sampleCount.Int32)
		c.SampleCount = &v
	}
	if observedAt.Valid {
		c.ObservedAt = observedAt.Time
	}
}

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

const pqUniqueViolation = "23505"

type ParcelPG struct {
	db     *sql.DB
	logger *zerolog.Logger
}

func NewParcelPG(db *sql.DB, logger *zerolog.Logger) *ParcelPG {
	return &ParcelPG{db: db, logger: logger}
}

func (r *ParcelPG) Create(ctx context.Context, p *entity.Parcel) error {
	snapshotJSON, err := json.Marshal(p.EGISSSnapshot)
	if err != nil {
		return fmt.Errorf("marshalling egiss snapshot: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `
		INSERT INTO parcels (
			cadastral_number, owner_wallet, area_ha, land_class,
			kyc_verified, oblast, rayon, holder_name, holder_iin_hash, egiss_snapshot,
			latitude, longitude
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, registered_at, updated_at`,
		p.CadastralNumber, p.OwnerWallet, p.AreaHa, p.LandClass,
		p.KYCVerified, p.Oblast, p.Rayon, p.HolderName, p.HolderIINHash, snapshotJSON,
		p.Latitude, p.Longitude,
	).Scan(&p.ID, &p.RegisteredAt, &p.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqUniqueViolation {
			return entity.ErrAlreadyExists
		}
		return fmt.Errorf("inserting parcel: %w", err)
	}

	return nil
}

func (r *ParcelPG) GetByCadastral(ctx context.Context, cadastral string) (*entity.Parcel, error) {
	var (
		p              entity.Parcel
		snapshotRaw    []byte
		onChain        sql.NullString
		oblast         sql.NullString
		rayon          sql.NullString
		holderName     sql.NullString
		holderIINHash  sql.NullString
	)

	err := r.db.QueryRowContext(ctx, `
		SELECT id, cadastral_number, owner_wallet, on_chain_address,
		       area_ha, land_class, kyc_verified, oblast, rayon,
		       holder_name, holder_iin_hash, egiss_snapshot,
		       registered_at, updated_at
		FROM parcels WHERE cadastral_number = $1`, cadastral,
	).Scan(
		&p.ID, &p.CadastralNumber, &p.OwnerWallet, &onChain,
		&p.AreaHa, &p.LandClass, &p.KYCVerified, &oblast, &rayon,
		&holderName, &holderIINHash, &snapshotRaw,
		&p.RegisteredAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("querying parcel by cadastral %q: %w", cadastral, err)
	}

	p.OnChainAddress = onChain.String
	p.Oblast = oblast.String
	p.Rayon = rayon.String
	p.HolderName = holderName.String
	p.HolderIINHash = holderIINHash.String

	if len(snapshotRaw) > 0 {
		_ = json.Unmarshal(snapshotRaw, &p.EGISSSnapshot)
	}

	return &p, nil
}

func (r *ParcelPG) ListNeedingSeasonalCheck(ctx context.Context, maxAge time.Duration) ([]entity.Parcel, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, cadastral_number, owner_wallet, on_chain_address, area_ha, land_class
		FROM parcels
		WHERE on_chain_address IS NOT NULL
		  AND (last_seasonal_check IS NULL OR last_seasonal_check < NOW() - $1::interval)
		LIMIT 50`, maxAge.String())
	if err != nil {
		return nil, fmt.Errorf("listing parcels needing seasonal check: %w", err)
	}
	defer rows.Close()

	var parcels []entity.Parcel
	for rows.Next() {
		var p entity.Parcel
		var onChain sql.NullString
		if err := rows.Scan(&p.ID, &p.CadastralNumber, &p.OwnerWallet, &onChain, &p.AreaHa, &p.LandClass); err != nil {
			return nil, fmt.Errorf("scanning parcel row: %w", err)
		}
		p.OnChainAddress = onChain.String
		parcels = append(parcels, p)
	}

	return parcels, rows.Err()
}

func (r *ParcelPG) ListAll(ctx context.Context) ([]entity.Parcel, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.cadastral_number, p.owner_wallet, p.on_chain_address,
		       p.area_ha, p.land_class, p.oblast, p.latitude, p.longitude
		FROM parcels p
		ORDER BY p.registered_at DESC
		LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("listing all parcels: %w", err)
	}
	defer rows.Close()

	var parcels []entity.Parcel
	for rows.Next() {
		var p entity.Parcel
		var onChain, oblast sql.NullString
		var lat, lon sql.NullFloat64
		if err := rows.Scan(
			&p.ID, &p.CadastralNumber, &p.OwnerWallet, &onChain,
			&p.AreaHa, &p.LandClass, &oblast, &lat, &lon,
		); err != nil {
			return nil, fmt.Errorf("scanning parcel row: %w", err)
		}
		p.OnChainAddress = onChain.String
		p.Oblast = oblast.String
		if lat.Valid {
			p.Latitude = &lat.Float64
		}
		if lon.Valid {
			p.Longitude = &lon.Float64
		}
		parcels = append(parcels, p)
	}

	return parcels, rows.Err()
}

func (r *ParcelPG) UpdateOnChainAddress(ctx context.Context, cadastral, addr string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE parcels SET on_chain_address = $1, updated_at = NOW()
		WHERE cadastral_number = $2`, addr, cadastral)
	if err != nil {
		return fmt.Errorf("updating on-chain address: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return entity.ErrNotFound
	}

	return nil
}

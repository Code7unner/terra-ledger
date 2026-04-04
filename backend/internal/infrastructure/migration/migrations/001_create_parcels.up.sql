CREATE TABLE parcels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cadastral_number VARCHAR(20) UNIQUE NOT NULL,
    owner_wallet VARCHAR(44) NOT NULL,
    on_chain_address VARCHAR(44),
    area_ha NUMERIC(10,2) NOT NULL,
    land_class SMALLINT NOT NULL CHECK (land_class BETWEEN 1 AND 8),
    kyc_verified BOOLEAN DEFAULT FALSE,
    oblast VARCHAR(50),
    rayon VARCHAR(50),
    holder_name VARCHAR(100),
    holder_iin_hash VARCHAR(64),
    egiss_snapshot JSONB,
    registered_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_parcels_cadastral ON parcels(cadastral_number);
CREATE INDEX idx_parcels_wallet ON parcels(owner_wallet);

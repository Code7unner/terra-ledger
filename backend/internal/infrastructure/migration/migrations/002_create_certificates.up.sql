CREATE TABLE certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id UUID NOT NULL REFERENCES parcels(id),
    cadastral_number VARCHAR(20) NOT NULL,
    season VARCHAR(10) NOT NULL,
    ndvi_score NUMERIC(5,3) NOT NULL,
    crop_type VARCHAR(50),
    yield_t_ha NUMERIC(6,2),
    sentinel_scene_id VARCHAR(100),
    on_chain_address VARCHAR(44),
    tx_signature VARCHAR(88),
    minted_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_certs_parcel ON certificates(parcel_id);
CREATE INDEX idx_certs_cadastral_season ON certificates(cadastral_number, season);

CREATE TABLE liens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id UUID NOT NULL REFERENCES parcels(id),
    cadastral_number VARCHAR(20) NOT NULL,
    lender_wallet VARCHAR(44) NOT NULL,
    lender_name VARCHAR(100),
    amount_tenge BIGINT NOT NULL,
    notary_cert_hash VARCHAR(64),
    on_chain_address VARCHAR(44),
    tx_signature VARCHAR(88),
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'released', 'disputed')),
    registered_at TIMESTAMPTZ DEFAULT NOW(),
    released_at TIMESTAMPTZ
);

CREATE INDEX idx_liens_parcel ON liens(parcel_id);
CREATE INDEX idx_liens_cadastral_active ON liens(cadastral_number) WHERE status = 'active';

CREATE TABLE consents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_address TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending',
    granted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE consent_access_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consent_id UUID NOT NULL REFERENCES consents(id),
    lender_wallet TEXT NOT NULL,
    lender_name TEXT,
    data_type TEXT NOT NULL,
    accessed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_consent_access_log_consent ON consent_access_log(consent_id);

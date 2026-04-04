CREATE TABLE credit_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id UUID NOT NULL REFERENCES parcels(id) UNIQUE,
    cadastral_number VARCHAR(20) UNIQUE NOT NULL,
    ai_score SMALLINT CHECK (ai_score BETWEEN 0 AND 100),
    recommended_ltv NUMERIC(4,3),
    collateral_grade CHAR(1) CHECK (collateral_grade IN ('A', 'B', 'C', 'D')),
    estimated_value_tenge BIGINT,
    model_version VARCHAR(20),
    explanation TEXT,
    risk_factors JSONB,
    computed_at TIMESTAMPTZ DEFAULT NOW()
);

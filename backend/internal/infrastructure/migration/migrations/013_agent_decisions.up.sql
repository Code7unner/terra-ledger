CREATE TABLE agent_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id UUID REFERENCES parcels(id),
    cadastral_number VARCHAR(20) NOT NULL,
    previous_score SMALLINT,
    new_score SMALLINT NOT NULL,
    previous_grade CHAR(1),
    new_grade CHAR(1) NOT NULL,
    reason TEXT NOT NULL,
    tx_signature VARCHAR(88),
    decided_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agent_decisions_cadastral ON agent_decisions(cadastral_number);
CREATE INDEX idx_agent_decisions_decided ON agent_decisions(decided_at DESC);

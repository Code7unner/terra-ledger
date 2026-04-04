CREATE TABLE lenders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    bin VARCHAR(12),
    api_key VARCHAR(64) UNIQUE NOT NULL,
    tier VARCHAR(20) DEFAULT 'starter'
        CHECK (tier IN ('starter', 'standard', 'enterprise', 'api_only')),
    queries_this_month INT DEFAULT 0,
    query_limit INT DEFAULT 200,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

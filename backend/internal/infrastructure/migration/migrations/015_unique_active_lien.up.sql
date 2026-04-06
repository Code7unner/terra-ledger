CREATE UNIQUE INDEX IF NOT EXISTS idx_liens_unique_active_cadastral
    ON liens (cadastral_number)
    WHERE status = 'active';

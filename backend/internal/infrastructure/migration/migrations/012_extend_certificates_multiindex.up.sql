ALTER TABLE certificates
  ADD COLUMN IF NOT EXISTS ndwi_score NUMERIC(6,4),
  ADD COLUMN IF NOT EXISTS evi_score NUMERIC(6,4),
  ADD COLUMN IF NOT EXISTS lai_estimate NUMERIC(6,4),
  ADD COLUMN IF NOT EXISTS cloud_free_pct NUMERIC(5,2),
  ADD COLUMN IF NOT EXISTS sample_count INTEGER,
  ADD COLUMN IF NOT EXISTS observed_at TIMESTAMPTZ;

UPDATE certificates SET observed_at = minted_at WHERE observed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_certs_cadastral_observed ON certificates(cadastral_number, observed_at DESC);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'uq_cert_cadastral_observed'
  ) THEN
    ALTER TABLE certificates ADD CONSTRAINT uq_cert_cadastral_observed UNIQUE (cadastral_number, observed_at);
  END IF;
END$$;

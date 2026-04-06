DROP INDEX IF EXISTS idx_certs_cadastral_observed;
ALTER TABLE certificates DROP CONSTRAINT IF EXISTS uq_cert_cadastral_observed;
ALTER TABLE certificates
  DROP COLUMN IF EXISTS ndwi_score,
  DROP COLUMN IF EXISTS evi_score,
  DROP COLUMN IF EXISTS lai_estimate,
  DROP COLUMN IF EXISTS cloud_free_pct,
  DROP COLUMN IF EXISTS sample_count,
  DROP COLUMN IF EXISTS observed_at;

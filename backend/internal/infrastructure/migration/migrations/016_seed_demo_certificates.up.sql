-- Seed NDVI certificates for demo parcels (multi-index: NDVI, NDWI, EVI, LAI)
INSERT INTO certificates (parcel_id, cadastral_number, season, ndvi_score, ndwi_score, evi_score, lai_estimate, cloud_free_pct, sample_count, crop_type, yield_t_ha, observed_at, minted_at)
SELECT p.id, p.cadastral_number, s.season, s.ndvi, s.ndwi, s.evi, s.lai, s.cloud_free, s.samples, s.crop, s.yield_t, s.obs, s.obs
FROM parcels p
JOIN (VALUES
    -- KZ11-0033-001: strong parcel, improving trend
    ('KZ11-0033-001', '2025-Q2', 0.72, -0.08, 0.45, 3.2, 89.5, 12, 'wheat', 2.8, '2025-04-15'::timestamptz),
    ('KZ11-0033-001', '2025-Q3', 0.76, -0.05, 0.48, 3.5, 92.1, 15, 'wheat', 3.1, '2025-07-15'::timestamptz),
    ('KZ11-0033-001', '2025-Q4', 0.74, -0.12, 0.44, 3.1, 85.3, 11, 'wheat', 2.9, '2025-10-15'::timestamptz),
    ('KZ11-0033-001', '2026-Q1', 0.78, -0.03, 0.50, 3.8, 91.0, 14, 'wheat', 3.3, '2026-01-15'::timestamptz),

    -- KZ11-0033-002: moderate parcel, stable
    ('KZ11-0033-002', '2025-Q2', 0.65, -0.18, 0.38, 2.5, 82.0, 10, 'barley', 2.2, '2025-04-15'::timestamptz),
    ('KZ11-0033-002', '2025-Q3', 0.68, -0.15, 0.40, 2.7, 88.4, 13, 'barley', 2.4, '2025-07-15'::timestamptz),
    ('KZ11-0033-002', '2025-Q4', 0.63, -0.22, 0.36, 2.3, 79.1, 9,  'barley', 2.1, '2025-10-15'::timestamptz),
    ('KZ11-0033-002', '2026-Q1', 0.66, -0.16, 0.39, 2.6, 84.7, 11, 'barley', 2.3, '2026-01-15'::timestamptz),

    -- KZ11-0033-003: excellent parcel, high productivity
    ('KZ11-0033-003', '2025-Q2', 0.80, 0.02,  0.55, 4.2, 95.0, 18, 'wheat', 3.8, '2025-04-15'::timestamptz),
    ('KZ11-0033-003', '2025-Q3', 0.82, 0.05,  0.58, 4.5, 96.2, 20, 'wheat', 4.0, '2025-07-15'::timestamptz),
    ('KZ11-0033-003', '2025-Q4', 0.79, -0.01, 0.53, 4.0, 93.1, 16, 'wheat', 3.7, '2025-10-15'::timestamptz),
    ('KZ11-0033-003', '2026-Q1', 0.83, 0.04,  0.57, 4.6, 94.8, 19, 'wheat', 4.1, '2026-01-15'::timestamptz)
) AS s(cadastral, season, ndvi, ndwi, evi, lai, cloud_free, samples, crop, yield_t, obs)
ON p.cadastral_number = s.cadastral
ON CONFLICT ON CONSTRAINT uq_cert_cadastral_observed DO NOTHING;

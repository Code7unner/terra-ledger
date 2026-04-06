INSERT INTO credit_scores (parcel_id, cadastral_number, ai_score, recommended_ltv, collateral_grade, estimated_value_tenge, model_version, explanation, risk_factors)
SELECT id, cadastral_number,
    CASE cadastral_number
        WHEN 'KZ11-0032-001' THEN 75
        WHEN 'KZ11-0032-002' THEN 65
        WHEN 'KZ11-0032-003' THEN 95
    END,
    CASE cadastral_number
        WHEN 'KZ11-0032-001' THEN 0.500
        WHEN 'KZ11-0032-002' THEN 0.500
        WHEN 'KZ11-0032-003' THEN 0.700
    END,
    CASE cadastral_number
        WHEN 'KZ11-0032-001' THEN 'B'
        WHEN 'KZ11-0032-002' THEN 'B'
        WHEN 'KZ11-0032-003' THEN 'A'
    END,
    CASE cadastral_number
        WHEN 'KZ11-0032-001' THEN 75000000
        WHEN 'KZ11-0032-002' THEN 110250000
        WHEN 'KZ11-0032-003' THEN 42500000
    END,
    'fallback-v1',
    CASE cadastral_number
        WHEN 'KZ11-0032-001' THEN 'Good productivity parcel in Akmola region. No active liens, land class 2.'
        WHEN 'KZ11-0032-002' THEN 'Average productivity parcel in Akmola region. No active liens, land class 3.'
        WHEN 'KZ11-0032-003' THEN 'High productivity parcel in Akmola region. No active liens, top land class.'
    END,
    '[]'::jsonb
FROM parcels
WHERE cadastral_number IN ('KZ11-0032-001', 'KZ11-0032-002', 'KZ11-0032-003')
ON CONFLICT (cadastral_number) DO NOTHING;

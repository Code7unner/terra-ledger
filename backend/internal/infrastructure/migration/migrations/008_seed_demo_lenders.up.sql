INSERT INTO lenders (name, bin, api_key, tier, query_limit) VALUES
    ('Halyk Bank',   '930740000137', 'tl_halyk_demo_2026_enterprise_key',   'enterprise', 999999),
    ('Kaspi',        '971240001315', 'tl_kaspi_demo_2026_standard_key',     'standard',   500),
    ('Bereke Bank',  '050540004758', 'tl_bereke_demo_2026_starter_key',     'starter',    200)
ON CONFLICT (api_key) DO NOTHING;

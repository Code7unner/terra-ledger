INSERT INTO parcels (cadastral_number, owner_wallet, area_ha, land_class, kyc_verified, oblast, rayon, holder_name, holder_iin_hash, egiss_snapshot) VALUES
    ('KZ11-0032-001', 'Gk7vSsbuMQ4YXEqTNdS8MjbKjPfruyVaFSfgBhAPo3rN', 150.00, 2, TRUE, 'Akmola', 'Shortandy', 'Askar Omarov',    '8a3b1f...mock', '{"source":"egiss_mock","verified":true}'),
    ('KZ11-0032-002', 'F4J6gPtHvQuR2e8mLKwAs3Y7NxZrE9fDcXbW5uTpV1hS', 220.50, 3, TRUE, 'Akmola', 'Shortandy', 'Bolat Tulegenov', '9c4d2e...mock', '{"source":"egiss_mock","verified":true}'),
    ('KZ11-0032-003', 'H9mNkR3wYqLdT7vXpC5sE2jA8fBuGx4ZoK6iWrM1hVnQ', 85.00,  1, TRUE, 'Akmola', 'Shortandy', 'Dana Serikova',   'ab5e3f...mock', '{"source":"egiss_mock","verified":true}')
ON CONFLICT (cadastral_number) DO NOTHING;

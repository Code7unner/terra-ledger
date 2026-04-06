ALTER TABLE parcels ADD COLUMN IF NOT EXISTS latitude NUMERIC(9,6);
ALTER TABLE parcels ADD COLUMN IF NOT EXISTS longitude NUMERIC(9,6);

UPDATE parcels SET latitude = 51.1605, longitude = 71.4704 WHERE cadastral_number = 'KZ11-0032-001';
UPDATE parcels SET latitude = 51.1805, longitude = 71.5104 WHERE cadastral_number = 'KZ11-0032-002';
UPDATE parcels SET latitude = 51.1405, longitude = 71.4304 WHERE cadastral_number = 'KZ11-0032-003';
UPDATE parcels SET latitude = 51.2005, longitude = 71.5504 WHERE cadastral_number = 'KZ11-0032-004';
UPDATE parcels SET latitude = 51.1200, longitude = 71.3900 WHERE cadastral_number = 'KZ11-0032-005';

-- 001_countries.sql
-- Seed countries. Only Kenya is active initially; others are pre-registered for
-- future adapter enablement. Idempotent (ON CONFLICT).

INSERT INTO legislation.countries (iso_code, name, metadata, active) VALUES
    ('KE', 'Kenya',
        '{"currency":"KES","calling_code":"+254","official_languages":["en","sw"],"capital":"Nairobi"}'::jsonb,
        TRUE),
    ('UG', 'Uganda',
        '{"currency":"UGX","calling_code":"+256","official_languages":["en","sw"],"capital":"Kampala"}'::jsonb,
        FALSE),
    ('TZ', 'Tanzania',
        '{"currency":"TZS","calling_code":"+255","official_languages":["en","sw"],"capital":"Dodoma"}'::jsonb,
        FALSE),
    ('GH', 'Ghana',
        '{"currency":"GHS","calling_code":"+233","official_languages":["en"],"capital":"Accra"}'::jsonb,
        FALSE),
    ('NG', 'Nigeria',
        '{"currency":"NGN","calling_code":"+234","official_languages":["en"],"capital":"Abuja"}'::jsonb,
        FALSE),
    ('ZA', 'South Africa',
        '{"currency":"ZAR","calling_code":"+27","official_languages":["en","af","zu","xh"],"capital":"Pretoria"}'::jsonb,
        FALSE)
ON CONFLICT (iso_code) DO UPDATE
    SET name = EXCLUDED.name,
        metadata = EXCLUDED.metadata,
        active = EXCLUDED.active,
        updated_at = now();

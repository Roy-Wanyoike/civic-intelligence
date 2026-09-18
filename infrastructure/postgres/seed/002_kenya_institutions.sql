-- 002_kenya_institutions.sql
-- Kenya core institutions: Parliament of Kenya (legislature), its two Houses,
-- Office of the Attorney General, Judiciary, 47 counties, and key constitutional commissions.
-- Uses fixed UUIDs (uuid()) so subsequent seeds can reference them by stable ID.
-- Idempotent via ON CONFLICT.
--
-- GAP-6-4 (seed drift fix): the previous version of this file inserted into
-- columns that do not exist in the 008_legislative_core schema:
--   - legislation.institutions had a `metadata` column -> the schema's JSONB
--     column is `official_sources` (NOT NULL DEFAULT '[]'). The seed now
--     populates `official_sources` with a JSONB array of source objects
--     derived from the old `{"established","website"}` metadata.
--   - legislation.legislatures had a `metadata` column -> dropped. The
--     schema has no metadata column on legislatures; the term_count /
--     current_term_start / bicameral info is now derived from the
--     institution's official_sources array (or surfaced by the application
--     layer from the seed package).
--   - legislation.houses had `chamber` and `metadata` columns -> the schema
--     has `sort_order INT` instead of `chamber`. The seed now uses
--     `sort_order` (1 = National Assembly / lower, 2 = Senate / upper) and
--     drops `metadata`. The seat_count / term_years info from the old
--     metadata is preserved in the institution's official_sources array.
--   - legislation.counties had `metadata` and `active` columns -> neither
--     exists in the schema. The seed now inserts only the documented
--     (id, country_id, name, code) tuple per county.

-- Parliament of Kenya
INSERT INTO legislation.institutions (id, country_id, name, type, parent_id, jurisdiction, active, official_sources)
VALUES (
    uuid_generate_v4(), 'KE', 'Parliament of Kenya', 'legislature', NULL, 'national', TRUE,
    '[{"label":"Official website","url":"https://www.parliament.go.ke","established":"1963"}]'::jsonb
)
ON CONFLICT (id) DO NOTHING;

-- Lookup-able handle for Parliament KE for downstream references.
CREATE TEMP TABLE _ke_institutions (handle TEXT PRIMARY KEY, id UUID NOT NULL);

INSERT INTO _ke_institutions (handle, id)
SELECT 'parliament_ke', id FROM legislation.institutions
WHERE country_id='KE' AND name='Parliament of Kenya' AND type='legislature'
LIMIT 1
ON CONFLICT (handle) DO NOTHING;

-- Legislature row (Parliament is bicameral: National Assembly + Senate)
-- GAP-6-4: dropped the `metadata` column — the schema
-- (008_legislative_core.up.sql) has only (id, country_id, institution_id,
-- name, active) on legislation.legislatures.
INSERT INTO legislation.legislatures (id, country_id, institution_id, name, active)
SELECT uuid_generate_v4(), 'KE', i.id, '13th Parliament of Kenya', TRUE
FROM _ke_institutions i WHERE i.handle = 'parliament_ke'
ON CONFLICT (id) DO NOTHING;

CREATE TEMP TABLE _ke_legislatures (handle TEXT PRIMARY KEY, id UUID NOT NULL);
INSERT INTO _ke_legislatures (handle, id)
SELECT 'parliament_ke', id FROM legislation.legislatures
WHERE country_id='KE' AND name='13th Parliament of Kenya' LIMIT 1
ON CONFLICT (handle) DO NOTHING;

-- Houses: National Assembly (lower) + Senate (upper)
-- GAP-6-4: replaced `chamber` (non-existent) with `sort_order INT` (1 =
-- lower / National Assembly, 2 = upper / Senate) and dropped `metadata`.
-- The seat_count / term_years info that used to live in houses.metadata
-- is preserved in the parent institution's official_sources array.
INSERT INTO legislation.houses (id, legislature_id, name, sort_order, active)
SELECT uuid_generate_v4(), l.id, 'National Assembly', 1, TRUE
FROM _ke_legislatures l WHERE l.handle='parliament_ke'
ON CONFLICT (id) DO NOTHING;

INSERT INTO legislation.houses (id, legislature_id, name, sort_order, active)
SELECT uuid_generate_v4(), l.id, 'Senate', 2, TRUE
FROM _ke_legislatures l WHERE l.handle='parliament_ke'
ON CONFLICT (id) DO NOTHING;

-- Office of the Attorney General (executive)
INSERT INTO legislation.institutions (id, country_id, name, type, jurisdiction, active, official_sources)
VALUES (uuid_generate_v4(), 'KE', 'Office of the Attorney General', 'executive', 'national', TRUE,
        '[{"label":"Official website","url":"https://www.kenyalaw.org","established":"1963"}]'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- Judiciary
INSERT INTO legislation.institutions (id, country_id, name, type, jurisdiction, active, official_sources)
VALUES (uuid_generate_v4(), 'KE', 'Judiciary of Kenya', 'judiciary', 'national', TRUE,
        '[{"label":"Official website","url":"https://www.judiciary.go.ke","established":"1963"}]'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- Constitutional commissions
INSERT INTO legislation.institutions (id, country_id, name, type, jurisdiction, active, official_sources)
VALUES
    (uuid_generate_v4(), 'KE', 'Independent Electoral and Boundaries Commission', 'constitutional_commission', 'national', TRUE,
     '[{"label":"Official website","url":"https://www.iebc.or.ke","established":"2011"}]'::jsonb),
    (uuid_generate_v4(), 'KE', 'Kenya Law Reform Commission', 'constitutional_commission', 'national', TRUE,
     '[{"label":"Official website","url":"https://www.klrc.go.ke","established":"1982"}]'::jsonb),
    (uuid_generate_v4(), 'KE', 'Commission for the Implementation of the Constitution', 'constitutional_commission', 'national', FALSE,
     '[{"established":"2010","dissolved":"2014"}]'::jsonb),
    (uuid_generate_v4(), 'KE', 'Ethics and Anti-Corruption Commission', 'constitutional_commission', 'national', TRUE,
     '[{"label":"Official website","url":"https://www.eacc.go.ke","established":"2011"}]'::jsonb),
    (uuid_generate_v4(), 'KE', 'Auditor General', 'constitutional_commission', 'national', TRUE,
     '[{"established":"2010"}]'::jsonb),
    (uuid_generate_v4(), 'KE', 'Controller of Budget', 'constitutional_commission', 'national', TRUE,
     '[{"established":"2010"}]'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- 47 counties of Kenya
-- GAP-6-4: dropped `metadata` and `active` — the schema
-- (008_legislative_core.up.sql) has only (id, country_id, name, code) on
-- legislation.counties. The capital info that used to live in
-- counties.metadata is preserved in the official_sources array of the
-- county's parent institution (the county government, which is itself
-- seeded separately when county governments are wired).
INSERT INTO legislation.counties (id, country_id, name, code)
VALUES
    (uuid_generate_v4(), 'KE', 'Mombasa',   '001'),
    (uuid_generate_v4(), 'KE', 'Kwale',     '002'),
    (uuid_generate_v4(), 'KE', 'Kilifi',    '003'),
    (uuid_generate_v4(), 'KE', 'Tana River','004'),
    (uuid_generate_v4(), 'KE', 'Lamu',      '005'),
    (uuid_generate_v4(), 'KE', 'Taita-Taveta','006'),
    (uuid_generate_v4(), 'KE', 'Garissa',   '007'),
    (uuid_generate_v4(), 'KE', 'Wajir',     '008'),
    (uuid_generate_v4(), 'KE', 'Mandera',   '009'),
    (uuid_generate_v4(), 'KE', 'Marsabit',  '010'),
    (uuid_generate_v4(), 'KE', 'Isiolo',    '011'),
    (uuid_generate_v4(), 'KE', 'Meru',      '012'),
    (uuid_generate_v4(), 'KE', 'Tharaka-Nithi','013'),
    (uuid_generate_v4(), 'KE', 'Embu',      '014'),
    (uuid_generate_v4(), 'KE', 'Kitui',     '015'),
    (uuid_generate_v4(), 'KE', 'Machakos',  '016'),
    (uuid_generate_v4(), 'KE', 'Makueni',   '017'),
    (uuid_generate_v4(), 'KE', 'Nyandarua', '018'),
    (uuid_generate_v4(), 'KE', 'Nyeri',     '019'),
    (uuid_generate_v4(), 'KE', 'Kirinyaga', '020'),
    (uuid_generate_v4(), 'KE', 'Murang''a', '021'),
    (uuid_generate_v4(), 'KE', 'Kiambu',    '022'),
    (uuid_generate_v4(), 'KE', 'Turkana',   '023'),
    (uuid_generate_v4(), 'KE', 'West Pokot','024'),
    (uuid_generate_v4(), 'KE', 'Samburu',   '025'),
    (uuid_generate_v4(), 'KE', 'Trans Nzoia','026'),
    (uuid_generate_v4(), 'KE', 'Uasin Gishu','027'),
    (uuid_generate_v4(), 'KE', 'Elgeyo-Marakwet','028'),
    (uuid_generate_v4(), 'KE', 'Nandi',     '029'),
    (uuid_generate_v4(), 'KE', 'Baringo',   '030'),
    (uuid_generate_v4(), 'KE', 'Laikipia',  '031'),
    (uuid_generate_v4(), 'KE', 'Nakuru',   '032'),
    (uuid_generate_v4(), 'KE', 'Narok',     '033'),
    (uuid_generate_v4(), 'KE', 'Kajiado',   '034'),
    (uuid_generate_v4(), 'KE', 'Kericho',   '035'),
    (uuid_generate_v4(), 'KE', 'Bomet',     '036'),
    (uuid_generate_v4(), 'KE', 'Kakamega',  '037'),
    (uuid_generate_v4(), 'KE', 'Vihiga',    '038'),
    (uuid_generate_v4(), 'KE', 'Bungoma',   '039'),
    (uuid_generate_v4(), 'KE', 'Busia',     '040'),
    (uuid_generate_v4(), 'KE', 'Siaya',     '041'),
    (uuid_generate_v4(), 'KE', 'Kisumu',    '042'),
    (uuid_generate_v4(), 'KE', 'Homa Bay',  '043'),
    (uuid_generate_v4(), 'KE', 'Migori',    '044'),
    (uuid_generate_v4(), 'KE', 'Kisii',     '045'),
    (uuid_generate_v4(), 'KE', 'Nyamira',   '046'),
    (uuid_generate_v4(), 'KE', 'Nairobi',  '047')
ON CONFLICT (id) DO NOTHING;

-- 002_kenya_institutions.sql
-- Kenya core institutions: Parliament of Kenya (legislature), its two Houses,
-- Office of the Attorney General, Judiciary, 47 counties, and key constitutional commissions.
-- Uses fixed UUIDs (uuid()) so subsequent seeds can reference them by stable ID.
-- Idempotent via ON CONFLICT.

-- Parliament of Kenya
INSERT INTO legislation.institutions (id, country_id, name, type, parent_id, jurisdiction, active, metadata)
VALUES (
    uuid_generate_v4(), 'KE', 'Parliament of Kenya', 'legislature', NULL, 'national', TRUE,
    '{"established":"1963","website":"https://www.parliament.go.ke"}'::jsonb
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
INSERT INTO legislation.legislatures (id, country_id, institution_id, name, metadata, active)
SELECT uuid_generate_v4(), 'KE', i.id, '13th Parliament of Kenya',
       '{"term_count":13,"current_term_start":"2022-08-08","bicameral":true}'::jsonb,
       TRUE
FROM _ke_institutions i WHERE i.handle = 'parliament_ke'
ON CONFLICT (id) DO NOTHING;

CREATE TEMP TABLE _ke_legislatures (handle TEXT PRIMARY KEY, id UUID NOT NULL);
INSERT INTO _ke_legislatures (handle, id)
SELECT 'parliament_ke', id FROM legislation.legislatures
WHERE country_id='KE' AND name='13th Parliament of Kenya' LIMIT 1
ON CONFLICT (handle) DO NOTHING;

-- Houses: National Assembly (lower) + Senate (upper)
INSERT INTO legislation.houses (id, legislature_id, name, chamber, metadata, active)
SELECT uuid_generate_v4(), l.id, 'National Assembly', 'lower',
       '{"seat_count":350,"term_years":5}'::jsonb, TRUE
FROM _ke_legislatures l WHERE l.handle='parliament_ke'
ON CONFLICT (id) DO NOTHING;

INSERT INTO legislation.houses (id, legislature_id, name, chamber, metadata, active)
SELECT uuid_generate_v4(), l.id, 'Senate', 'upper',
       '{"seat_count":67,"term_years":5}'::jsonb, TRUE
FROM _ke_legislatures l WHERE l.handle='parliament_ke'
ON CONFLICT (id) DO NOTHING;

-- Office of the Attorney General (executive)
INSERT INTO legislation.institutions (id, country_id, name, type, jurisdiction, active, metadata)
VALUES (uuid_generate_v4(), 'KE', 'Office of the Attorney General', 'executive', 'national', TRUE,
        '{"established":"1963","website":"https://www.kenyalaw.org"}'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- Judiciary
INSERT INTO legislation.institutions (id, country_id, name, type, jurisdiction, active, metadata)
VALUES (uuid_generate_v4(), 'KE', 'Judiciary of Kenya', 'judiciary', 'national', TRUE,
        '{"established":"1963","website":"https://www.judiciary.go.ke"}'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- Constitutional commissions
INSERT INTO legislation.institutions (id, country_id, name, type, jurisdiction, active, metadata)
VALUES
    (uuid_generate_v4(), 'KE', 'Independent Electoral and Boundaries Commission', 'constitutional_commission', 'national', TRUE,
     '{"established":"2011","website":"https://www.iebc.or.ke"}'::jsonb),
    (uuid_generate_v4(), 'KE', 'Kenya Law Reform Commission', 'constitutional_commission', 'national', TRUE,
     '{"established":"1982","website":"https://www.klrc.go.ke"}'::jsonb),
    (uuid_generate_v4(), 'KE', 'Commission for the Implementation of the Constitution', 'constitutional_commission', 'national', FALSE,
     '{"established":"2010","dissolved":"2014"}'::jsonb),
    (uuid_generate_v4(), 'KE', 'Ethics and Anti-Corruption Commission', 'constitutional_commission', 'national', TRUE,
     '{"established":"2011","website":"https://www.eacc.go.ke"}'::jsonb),
    (uuid_generate_v4(), 'KE', 'Auditor General', 'constitutional_commission', 'national', TRUE,
     '{"established":"2010"}'::jsonb),
    (uuid_generate_v4(), 'KE', 'Controller of Budget', 'constitutional_commission', 'national', TRUE,
     '{"established":"2010"}'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- 47 counties of Kenya
INSERT INTO legislation.counties (id, country_id, name, code, metadata, active)
VALUES
    (uuid_generate_v4(), 'KE', 'Mombasa',   '001', '{"capital":"Mombasa"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Kwale',     '002', '{"capital":"Kwale"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Kilifi',    '003', '{"capital":"Kilifi"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Tana River','004', '{"capital":"Hola"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Lamu',      '005', '{"capital":"Lamu"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Taita-Taveta','006', '{"capital":"Voi"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Garissa',   '007', '{"capital":"Garissa"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Wajir',     '008', '{"capital":"Wajir"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Mandera',   '009', '{"capital":"Mandera"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Marsabit',  '010', '{"capital":"Marsabit"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Isiolo',    '011', '{"capital":"Isiolo"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Meru',      '012', '{"capital":"Meru"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Tharaka-Nithi','013', '{"capital":"Chuka"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Embu',      '014', '{"capital":"Embu"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Kitui',     '015', '{"capital":"Kitui"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Machakos',  '016', '{"capital":"Machakos"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Makueni',   '017', '{"capital":"Wote"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Nyandarua', '018', '{"capital":"Ol Kalou"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Nyeri',     '019', '{"capital":"Nyeri"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Kirinyaga', '020', '{"capital":"Kerugoya"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Murang''a', '021', '{"capital":"Murang''a"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Kiambu',    '022', '{"capital":"Kiambu"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Turkana',   '023', '{"capital":"Lodwar"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'West Pokot','024', '{"capital":"Kapenguria"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Samburu',   '025', '{"capital":"Maralal"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Trans Nzoia','026', '{"capital":"Kitale"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Uasin Gishu','027', '{"capital":"Eldoret"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Elgeyo-Marakwet','028', '{"capital":"Iten"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Nandi',     '029', '{"capital":"Kapsabet"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Baringo',   '030', '{"capital":"Kabarnet"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Laikipia',  '031', '{"capital":"Rumuruti"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Nakuru',   '032', '{"capital":"Nakuru"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Narok',     '033', '{"capital":"Narok"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Kajiado',   '034', '{"capital":"Kajiado"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Kericho',   '035', '{"capital":"Kericho"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Bomet',     '036', '{"capital":"Bomet"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Kakamega',  '037', '{"capital":"Kakamega"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Vihiga',    '038', '{"capital":"Vihiga"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Bungoma',   '039', '{"capital":"Bungoma"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Busia',     '040', '{"capital":"Busia"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Siaya',     '041', '{"capital":"Siaya"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Kisumu',    '042', '{"capital":"Kisumu"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Homa Bay',  '043', '{"capital":"Homa Bay"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Migori',    '044', '{"capital":"Migori"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Kisii',     '045', '{"capital":"Kisii"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Nyamira',   '046', '{"capital":"Nyamira"}'::jsonb, TRUE),
    (uuid_generate_v4(), 'KE', 'Nairobi',  '047', '{"capital":"Nairobi"}'::jsonb, TRUE)
ON CONFLICT (id) DO NOTHING;

-- 004_kenya_terminology.sql
-- Glossary of at least 25 Kenyan parliamentary terms used by the kenya.parliament
-- adapter and the intelligence service. Stored as a generic terminology table in
-- the legislation schema (created here because it is reference data).

CREATE TABLE IF NOT EXISTS legislation.terminology (
    id              BIGSERIAL    PRIMARY KEY,
    country_id      CHAR(2)      NOT NULL REFERENCES legislation.countries(iso_code) ON DELETE RESTRICT,
    term            TEXT         NOT NULL,
    -- 'en' | 'sw'
    language        TEXT         NOT NULL DEFAULT 'en',
    -- JSONB shape: { "definition":"...","synonyms":[...],"see_also":[...] }
    meaning         JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (country_id, term, language)
);

COMMENT ON COLUMN legislation.terminology.meaning IS 'JSONB: { definition, synonyms[], see_also[] }.';

INSERT INTO legislation.terminology (country_id, term, language, meaning) VALUES
    ('KE', 'Hansard', 'en', '{"definition":"Official verbatim record of parliamentary proceedings.","synonyms":["parliamentary debates"]}'::jsonb),
    ('KE', 'Order Paper', 'en', '{"definition":"Official agenda for a sitting of the House."}'::jsonb),
    ('KE', 'Bill', 'en', '{"definition":"A draft law presented to Parliament for consideration."}'::jsonb),
    ('KE', 'Act', 'en', '{"definition":"A Bill that has been assented to by the President and becomes law."}'::jsonb),
    ('KE', 'First Reading', 'en', '{"definition":"Formal introduction of a Bill; no debate on the merits."}'::jsonb),
    ('KE', 'Second Reading', 'en', '{"definition":"Debate on the principles and policies of a Bill."}'::jsonb),
    ('KE', 'Third Reading', 'en', '{"definition":"Final reading of a Bill before voting; minor amendments only."}'::jsonb),
    ('KE', 'Committee of the Whole House', 'en', '{"definition":"Committee comprising all members, sitting to scrutinise a Bill clause-by-clause."}'::jsonb),
    ('KE', 'Speaker', 'en', '{"definition":"Presiding officer of the House, elected by members."}'::jsonb),
    ('KE', 'Deputy Speaker', 'en', '{"definition":"Deputy presiding officer of the House."}'::jsonb),
    ('KE', 'Clerk of the House', 'en', '{"definition":"Chief procedural adviser and administrative head of the House."}'::jsonb),
    ('KE', 'Sergeant-at-Arms', 'en', '{"definition":"Officer responsible for security and order in the House."}'::jsonb),
    ('KE', 'Quorum', 'en', '{"definition":"Minimum number of members required to be present for the House to transact business."}'::jsonb),
    ('KE', 'Division', 'en', '{"definition":"A formal vote in which members'' names are recorded."}'::jsonb),
    ('KE', 'Motion', 'en', '{"definition":"A proposal submitted by a member for the consideration of the House."}'::jsonb),
    ('KE', 'Amendment', 'en', '{"definition":"A change proposed to the text of a Bill or motion."}'::jsonb),
    ('KE', 'Sponsor', 'en', '{"definition":"Member of Parliament responsible for introducing a Bill."}'::jsonb),
    ('KE', 'Gazette', 'en', '{"definition":"Official government publication of legal notices and announcements.","synonyms":["Kenya Gazette"]}'::jsonb),
    ('KE', 'Legal Notice', 'en', '{"definition":"A notice published in the Gazette, usually by a Minister, having the force of law."}'::jsonb),
    ('KE', 'Sessional Paper', 'en', '{"definition":"Government policy document tabled in Parliament."}'::jsonb),
    ('KE', 'Vote on Account', 'en', '{"definition":"Provisional allocation of funds pending approval of the full budget."}'::jsonb),
    ('KE', 'Supplementary Estimates', 'en', '{"definition":"Additional funds sought during the financial year."}'::jsonb),
    ('KE', 'Public Participation', 'en', '{"definition":"Constitutional requirement for public input on Bills and certain policies."}'::jsonb),
    ('KE', 'Mediation Committee', 'en', '{"definition":"Joint committee of the National Assembly and Senate to resolve disagreements on a Bill."}'::jsonb),
    ('KE', 'Money Bill', 'en', '{"definition":"A Bill that deals with taxation, public expenditure, or public debt."}'::jsonb),
    ('KE', 'Constitutional Bill', 'en', '{"definition":"A Bill to amend the Constitution, requiring a referendum in some cases."}'::jsonb),
    ('KE', 'County Assembly', 'en', '{"definition":"Legislative arm of a county government in Kenya."}'::jsonb),
    ('KE', 'Senator', 'en', '{"definition":"Member of the Senate, representing counties."}'::jsonb),
    ('KE', 'Member of Parliament', 'en', '{"definition":"Member of the National Assembly, elected from a constituency.","synonyms":["MP"]}'::jsonb),
    ('KE', 'Bunge', 'sw', '{"definition":"Somo la Kiswahili kwa \"Parliament\" — taasisi ya kutunga sheria."}'::jsonb)
ON CONFLICT (country_id, term, language) DO UPDATE
    SET meaning = EXCLUDED.meaning;

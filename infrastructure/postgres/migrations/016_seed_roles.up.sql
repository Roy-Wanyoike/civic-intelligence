-- 016_seed_roles.up.sql
INSERT INTO identity.roles (code, description) VALUES
    ('citizen',    'Ordinary citizen — read access + follows + AI questions'),
    ('researcher', 'Researcher — read access + bulk export + API'),
    ('editor',     'Editor — review/accept candidate facts from AI'),
    ('admin',      'Administrator — full access')
ON CONFLICT (code) DO NOTHING;

INSERT INTO identity.permissions (code, description) VALUES
    ('bill:read',     'Read Bills and their timelines'),
    ('bill:follow',   'Follow Bills for notifications'),
    ('bill:question', 'Ask AI questions about Bills'),
    ('ai:stream',     'Stream AI responses via SSE'),
    ('search:query',  'Use the search API'),
    ('briefing:read', 'Read the daily civic brief'),
    ('editor:review', 'Review AI candidate facts'),
    ('admin:users',   'Manage users and roles'),
    ('admin:sources', 'Manage ingestion sources')
ON CONFLICT (code) DO NOTHING;

INSERT INTO identity.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM identity.roles r, identity.permissions p
WHERE
    (r.code = 'citizen'   AND p.code IN ('bill:read','bill:follow','bill:question','ai:stream','search:query','briefing:read'))
 OR (r.code = 'researcher' AND p.code IN ('bill:read','bill:follow','bill:question','ai:stream','search:query','briefing:read'))
 OR (r.code = 'editor'     AND p.code IN ('bill:read','bill:follow','bill:question','ai:stream','search:query','briefing:read','editor:review'))
 OR (r.code = 'admin'      AND TRUE)
ON CONFLICT DO NOTHING;

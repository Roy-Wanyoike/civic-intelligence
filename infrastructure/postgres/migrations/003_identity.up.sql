-- 003_identity.up.sql
CREATE TABLE identity.users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_oidc   TEXT NOT NULL UNIQUE,
    email           CITEXT NOT NULL UNIQUE,
    display_name    TEXT NOT NULL,
    locale          TEXT NOT NULL DEFAULT 'en-KE',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    is_admin        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE identity.sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    user_agent      TEXT,
    ip              INET
);
CREATE INDEX idx_sessions_user ON identity.sessions(user_id);

CREATE TABLE identity.roles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code            TEXT NOT NULL UNIQUE,
    description     TEXT
);

CREATE TABLE identity.permissions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code            TEXT NOT NULL UNIQUE,
    description     TEXT
);

CREATE TABLE identity.role_permissions (
    role_id         UUID NOT NULL REFERENCES identity.roles(id) ON DELETE CASCADE,
    permission_id   UUID NOT NULL REFERENCES identity.permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE identity.user_roles (
    user_id         UUID NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    role_id         UUID NOT NULL REFERENCES identity.roles(id) ON DELETE CASCADE,
    granted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by      UUID REFERENCES identity.users(id),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE identity.user_preferences (
    user_id             UUID PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
    preferred_country   TEXT NOT NULL DEFAULT 'KE',
    impact_lens         TEXT,
    ui_density          TEXT NOT NULL DEFAULT 'comfortable',
    email_digest        BOOLEAN NOT NULL DEFAULT TRUE,
    email_breaking      BOOLEAN NOT NULL DEFAULT TRUE,
    push_notifications  BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

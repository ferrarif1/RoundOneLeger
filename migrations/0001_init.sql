BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ip_allowlists (
    id SERIAL PRIMARY KEY,
    label TEXT NOT NULL,
    cidr TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL PRIMARY KEY,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT,
    hash TEXT NOT NULL,
    prev_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO users (username, password_hash, is_admin)
    VALUES ('hzdsz_admin', 'gLgPjUGkVuL1Pzwh5sM55w:cKnqW27kGVuQPr+sOHqC50e5TldcsLNFyaTTzAr+UnM', TRUE)
    ON CONFLICT (username) DO UPDATE
        SET password_hash = EXCLUDED.password_hash,
            is_admin = EXCLUDED.is_admin,
            updated_at = NOW();

COMMIT;

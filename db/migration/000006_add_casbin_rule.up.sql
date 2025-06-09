CREATE TABLE IF NOT EXISTS casbin_rule (
    id     SERIAL PRIMARY KEY,
    ptype  TEXT NOT NULL,
    v0     TEXT,
    v1     TEXT,
    v2     TEXT,
    v3     TEXT,
    v4     TEXT,
    v5     TEXT
);
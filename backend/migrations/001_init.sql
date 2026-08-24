-- GoServerless schema. Applied under pg_advisory_lock to avoid dual-migrate races.
CREATE TABLE IF NOT EXISTS functions (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL UNIQUE,
    runtime          TEXT NOT NULL,
    status           TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    timeout_sec      INTEGER NOT NULL DEFAULT 30,
    memory_mb        INTEGER NOT NULL DEFAULT 128,
    cpu_nano         BIGINT NOT NULL DEFAULT 500000000,
    max_concurrency  INTEGER NOT NULL DEFAULT 10,
    env_json         JSONB NOT NULL DEFAULT '{}'::jsonb,
    current_version  INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL,
    CONSTRAINT functions_name_chk CHECK (name ~ '^[a-z][a-z0-9-]{2,39}$'),
    CONSTRAINT functions_runtime_chk CHECK (runtime IN ('go', 'nodejs')),
    CONSTRAINT functions_status_chk CHECK (status IN ('DRAFT', 'BUILDING', 'READY', 'FAILED')),
    CONSTRAINT functions_timeout_chk CHECK (timeout_sec BETWEEN 1 AND 300),
    CONSTRAINT functions_memory_chk CHECK (memory_mb IN (64, 128, 256, 512))
);

CREATE TABLE IF NOT EXISTS function_versions (
    id             TEXT PRIMARY KEY,
    function_id    TEXT NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    version        INTEGER NOT NULL,
    status         TEXT NOT NULL,
    code           TEXT NOT NULL,
    artifact_path  TEXT NOT NULL DEFAULT '',
    build_log      TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL,
    UNIQUE (function_id, version)
);

CREATE INDEX IF NOT EXISTS idx_versions_fn ON function_versions (function_id, version DESC);

CREATE TABLE IF NOT EXISTS triggers (
    id           TEXT PRIMARY KEY,
    function_id  TEXT NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL,
    cron_expr    TEXT NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL,
    CONSTRAINT triggers_kind_chk CHECK (kind IN ('http', 'cron'))
);

CREATE INDEX IF NOT EXISTS idx_triggers_fn ON triggers (function_id);

CREATE TABLE IF NOT EXISTS invocations (
    id            TEXT PRIMARY KEY,
    function_id   TEXT NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    version       INTEGER NOT NULL,
    trigger_kind  TEXT NOT NULL,
    status_code   INTEGER NOT NULL,
    success       BOOLEAN NOT NULL,
    cold_start    BOOLEAN NOT NULL,
    wakeup_ms     BIGINT NOT NULL DEFAULT 0,
    exec_ms       BIGINT NOT NULL DEFAULT 0,
    e2e_ms        BIGINT NOT NULL DEFAULT 0,
    error         TEXT NOT NULL DEFAULT '',
    logs          TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_invocations_fn_time ON invocations (function_id, created_at DESC);

CREATE TABLE IF NOT EXISTS sessions (
    token       TEXT PRIMARY KEY,
    username    TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL
);

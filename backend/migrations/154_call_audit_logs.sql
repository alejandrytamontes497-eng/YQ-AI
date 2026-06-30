CREATE TABLE IF NOT EXISTS call_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    account_id BIGINT NULL REFERENCES accounts(id) ON DELETE SET NULL,
    group_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL,
    request_id VARCHAR(128) NOT NULL DEFAULT '',
    method VARCHAR(16) NOT NULL DEFAULT '',
    path VARCHAR(256) NOT NULL DEFAULT '',
    model VARCHAR(128) NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL DEFAULT 0,
    request_excerpt TEXT NOT NULL DEFAULT '',
    response_excerpt TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_call_audit_logs_user_created
    ON call_audit_logs (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_call_audit_logs_api_key_created
    ON call_audit_logs (api_key_id, created_at DESC);

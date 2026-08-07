CREATE TABLE IF NOT EXISTS image_generation_jobs (
    id VARCHAR(40) PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    model VARCHAR(128) NOT NULL,
    prompt TEXT NOT NULL,
    size VARCHAR(32) NOT NULL,
    quality VARCHAR(32) NOT NULL,
    image_count INTEGER NOT NULL DEFAULT 1,
    request_body JSONB NOT NULL,
    results JSONB NOT NULL DEFAULT '[]'::jsonb,
    error_message TEXT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT image_generation_jobs_status_check
        CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
    CONSTRAINT image_generation_jobs_count_check
        CHECK (image_count BETWEEN 1 AND 10)
);

CREATE INDEX IF NOT EXISTS idx_image_generation_jobs_user_created
    ON image_generation_jobs (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_image_generation_jobs_claim
    ON image_generation_jobs (status, created_at ASC);


DO $$ BEGIN RAISE NOTICE '[Migration 000075] Creating processing cache...'; END $$;

CREATE TABLE IF NOT EXISTS processing_cache_entries (
    tenant_id BIGINT NOT NULL,
    cache_type VARCHAR(32) NOT NULL,
    cache_key VARCHAR(64) NOT NULL,
    payload BYTEA NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    hit_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_hit_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (tenant_id, cache_type, cache_key)
);

CREATE INDEX IF NOT EXISTS idx_processing_cache_last_hit
    ON processing_cache_entries(last_hit_at);

DO $$ BEGIN RAISE NOTICE '[Migration 000075] Processing cache ready'; END $$;

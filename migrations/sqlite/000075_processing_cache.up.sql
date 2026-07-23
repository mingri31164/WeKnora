CREATE TABLE IF NOT EXISTS processing_cache_entries (
    tenant_id INTEGER NOT NULL,
    cache_type VARCHAR(32) NOT NULL,
    cache_key VARCHAR(64) NOT NULL,
    payload BLOB NOT NULL,
    size_bytes INTEGER NOT NULL DEFAULT 0,
    hit_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_hit_at DATETIME,
    PRIMARY KEY (tenant_id, cache_type, cache_key)
);

CREATE INDEX IF NOT EXISTS idx_processing_cache_last_hit
    ON processing_cache_entries(last_hit_at);

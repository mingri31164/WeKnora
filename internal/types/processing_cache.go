package types

import "time"

// ProcessingCacheEntry stores a content-addressed ingestion artifact.
// Cache keys are scoped by tenant and artifact type to prevent cross-tenant
// reuse of document-derived data.
type ProcessingCacheEntry struct {
	TenantID  uint64     `json:"tenant_id" gorm:"primaryKey;autoIncrement:false"`
	CacheType string     `json:"cache_type" gorm:"type:varchar(32);primaryKey"`
	CacheKey  string     `json:"cache_key" gorm:"type:varchar(64);primaryKey"`
	Payload   []byte     `json:"payload" gorm:"type:bytea;not null"`
	SizeBytes int64      `json:"size_bytes" gorm:"not null;default:0"`
	HitCount  int64      `json:"hit_count" gorm:"not null;default:0"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	LastHitAt *time.Time `json:"last_hit_at,omitempty" gorm:"index"`
}

func (ProcessingCacheEntry) TableName() string {
	return "processing_cache_entries"
}

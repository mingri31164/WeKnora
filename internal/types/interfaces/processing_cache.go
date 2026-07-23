package interfaces

import "context"

// ProcessingCacheRepository persists content-addressed ingestion artifacts.
// Implementations must isolate entries by tenant.
type ProcessingCacheRepository interface {
	Get(ctx context.Context, tenantID uint64, cacheType, cacheKey string) ([]byte, bool, error)
	Put(ctx context.Context, tenantID uint64, cacheType, cacheKey string, payload []byte) error
}

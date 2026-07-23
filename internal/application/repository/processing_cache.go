package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type processingCacheRepository struct {
	db *gorm.DB
}

func NewProcessingCacheRepository(db *gorm.DB) interfaces.ProcessingCacheRepository {
	return &processingCacheRepository{db: db}
}

func (r *processingCacheRepository) Get(
	ctx context.Context,
	tenantID uint64,
	cacheType, cacheKey string,
) ([]byte, bool, error) {
	var entry types.ProcessingCacheEntry
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND cache_type = ? AND cache_key = ?", tenantID, cacheType, cacheKey).
		First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	now := time.Now()
	_ = r.db.WithContext(ctx).Model(&types.ProcessingCacheEntry{}).
		Where("tenant_id = ? AND cache_type = ? AND cache_key = ?", tenantID, cacheType, cacheKey).
		Updates(map[string]any{
			"hit_count":   gorm.Expr("hit_count + 1"),
			"last_hit_at": now,
		}).Error

	return append([]byte(nil), entry.Payload...), true, nil
}

func (r *processingCacheRepository) Put(
	ctx context.Context,
	tenantID uint64,
	cacheType, cacheKey string,
	payload []byte,
) error {
	now := time.Now()
	entry := &types.ProcessingCacheEntry{
		TenantID:  tenantID,
		CacheType: cacheType,
		CacheKey:  cacheKey,
		Payload:   append([]byte(nil), payload...),
		SizeBytes: int64(len(payload)),
		CreatedAt: now,
		UpdatedAt: now,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "cache_type"},
			{Name: "cache_key"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"payload":    entry.Payload,
			"size_bytes": entry.SizeBytes,
			"updated_at": now,
		}),
	}).Create(entry).Error
}

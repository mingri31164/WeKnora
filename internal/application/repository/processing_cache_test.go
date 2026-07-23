package repository

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProcessingCacheRepositoryTenantIsolationAndUpsert(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:processing-cache?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ProcessingCacheEntry{}))

	repo := NewProcessingCacheRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Put(ctx, 1, "embedding", "key", []byte("tenant-1")))
	require.NoError(t, repo.Put(ctx, 2, "embedding", "key", []byte("tenant-2")))

	got, found, err := repo.Get(ctx, 1, "embedding", "key")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []byte("tenant-1"), got)

	got, found, err = repo.Get(ctx, 2, "embedding", "key")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []byte("tenant-2"), got)

	require.NoError(t, repo.Put(ctx, 1, "embedding", "key", []byte("updated")))
	got, found, err = repo.Get(ctx, 1, "embedding", "key")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []byte("updated"), got)

	_, found, err = repo.Get(ctx, 1, "embedding", "missing")
	require.NoError(t, err)
	require.False(t, found)

	var entry types.ProcessingCacheEntry
	require.NoError(t, db.Where(
		"tenant_id = ? AND cache_type = ? AND cache_key = ?", 1, "embedding", "key",
	).First(&entry).Error)
	require.EqualValues(t, 2, entry.HitCount)
	require.NotNil(t, entry.LastHitAt)
}

package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSQLiteMigrationsCreateProcessingCache(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(repoRoot))
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	dbPath := filepath.Join(t.TempDir(), "migration.db")
	require.NoError(t, RunMigrationsWithOptions(
		"sqlite3://"+dbPath,
		MigrationOptions{SQLiteDBPath: dbPath},
	))

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	var tableName string
	require.NoError(t, db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'processing_cache_entries'",
	).Scan(&tableName))
	require.Equal(t, "processing_cache_entries", tableName)

	version, dirty, available := CachedMigrationVersion()
	require.True(t, available)
	require.False(t, dirty)
	require.EqualValues(t, 75, version)
}

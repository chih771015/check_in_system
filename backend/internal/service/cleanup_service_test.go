package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"translator-checkin/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTempUploadDir swaps config.AppConfig.UploadDir for a fresh temp dir for
// the duration of one test, restoring the original value after.
func withTempUploadDir(t *testing.T, retentionDays int) string {
	t.Helper()
	initTestConfig()
	dir := t.TempDir()

	prev := *config.AppConfig
	t.Cleanup(func() { *config.AppConfig = prev })

	config.AppConfig.UploadDir = dir
	config.AppConfig.PhotoRetentionDays = retentionDays
	return dir
}

func touchFile(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	require.NoError(t, os.Chtimes(path, modTime, modTime))
}

func TestCleanupService_RemovesOldPhotos(t *testing.T) {
	dir := withTempUploadDir(t, 30)
	old := filepath.Join(dir, "old.jpg")
	fresh := filepath.Join(dir, "fresh.jpg")
	touchFile(t, old, time.Now().AddDate(0, 0, -60))
	touchFile(t, fresh, time.Now().AddDate(0, 0, -1))

	NewCleanupService().RunPhotoCleanup()

	_, errOld := os.Stat(old)
	assert.True(t, os.IsNotExist(errOld), "old photo should be removed")
	_, errFresh := os.Stat(fresh)
	assert.NoError(t, errFresh, "fresh photo should remain")
}

func TestCleanupService_SpareNonImageFiles(t *testing.T) {
	dir := withTempUploadDir(t, 30)
	old := filepath.Join(dir, "old.txt")
	touchFile(t, old, time.Now().AddDate(0, 0, -60))

	NewCleanupService().RunPhotoCleanup()

	_, err := os.Stat(old)
	assert.NoError(t, err, "non-image files (txt) should not be deleted")
}

func TestCleanupService_NoopWhenUploadDirMissing(t *testing.T) {
	initTestConfig()
	prev := *config.AppConfig
	t.Cleanup(func() { *config.AppConfig = prev })
	config.AppConfig.UploadDir = ""

	// Should not panic
	NewCleanupService().RunPhotoCleanup()
}

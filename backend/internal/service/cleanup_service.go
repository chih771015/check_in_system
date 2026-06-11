package service

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"translator-checkin/internal/config"
)

// CleanupService removes old photo uploads that are no longer referenced.
type CleanupService struct{}

// NewCleanupService creates a new CleanupService.
func NewCleanupService() *CleanupService {
	return &CleanupService{}
}

// RunPhotoCleanup walks the upload directory and removes files older than
// PhotoRetentionDays. It logs the count of deleted files.
func (c *CleanupService) RunPhotoCleanup() {
	cfg := config.AppConfig
	if cfg == nil || cfg.UploadDir == "" {
		return
	}
	retentionDays := cfg.PhotoRetentionDays
	// retentionDays <= 0 means permanent storage: never delete anything.
	// This is the production default — some certificates must be kept for
	// 5+ years, so automatic pruning is disabled.
	if retentionDays <= 0 {
		log.Println("[cleanup] PHOTO_RETENTION_DAYS <= 0 — permanent retention, skipping cleanup")
		return
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	deleted := 0
	err := filepath.Walk(cfg.UploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		// Only delete image-like files to avoid nuking unrelated content.
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			if rmErr := os.Remove(path); rmErr == nil {
				deleted++
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("[cleanup] walk error: %v", err)
	}
	log.Printf("[cleanup] removed %d photos older than %d days", deleted, retentionDays)
}

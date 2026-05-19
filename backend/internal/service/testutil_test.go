package service

import (
	"testing"

	"translator-checkin/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestDB returns an in-memory SQLite database with all relevant models
// migrated. Each call returns a fresh DB so tests are isolated.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Schedule{},
		&model.Checkin{},
		&model.ExportSchedule{},
		&model.AuditLog{},
		&model.Patient{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

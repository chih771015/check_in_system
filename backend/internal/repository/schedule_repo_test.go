package repository

import (
	"testing"
	"time"

	"translator-checkin/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newScheduleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Patient{}, &model.Schedule{}, &model.SchedulePatient{},
	))
	return db
}

func TestScheduleRepo_FindRecentByCreated_OrdersAndLimits(t *testing.T) {
	db := newScheduleTestDB(t)
	repo := NewScheduleRepository(db)
	date := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	base := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	// Insert three schedules with explicit (out-of-order) created_at timestamps.
	for _, ca := range []time.Time{
		base.Add(2 * time.Hour), // newest
		base,                    // oldest
		base.Add(1 * time.Hour), // middle
	} {
		s := &model.Schedule{
			TranslatorID: 0, Date: date,
			StartTime: "09:00", EndTime: "12:00", Location: "L",
			CreatedAt: ca,
		}
		require.NoError(t, db.Create(s).Error)
	}

	got, total, err := repo.FindRecentByCreated(1, 2)
	require.NoError(t, err)
	require.Len(t, got, 2, "page size should cap results")
	assert.Equal(t, int64(3), total, "total reflects full count regardless of page size")
	assert.True(t, got[0].CreatedAt.After(got[1].CreatedAt), "newest created_at first (DESC)")

	// Second page returns the remaining row.
	page2, _, err := repo.FindRecentByCreated(2, 2)
	require.NoError(t, err)
	require.Len(t, page2, 1, "page 2 has the leftover row")
}

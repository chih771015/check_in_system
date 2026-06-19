package service

import (
	"context"
	"testing"
	"time"

	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonthRange(t *testing.T) {
	from, to, label := monthRange(time.Date(2026, 6, 19, 10, 30, 0, 0, time.UTC))
	assert.Equal(t, "2026-06-01", from)
	assert.Equal(t, "2026-07-01", to)
	assert.Equal(t, "2026-06", label)

	// December rolls the exclusive upper bound into the next year.
	from, to, label = monthRange(time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, "2026-12-01", from)
	assert.Equal(t, "2027-01-01", to)
	assert.Equal(t, "2026-12", label)
}

func TestStatsService_CurrentMonthActualTotal(t *testing.T) {
	db := newTestDB(t)
	spRepo := repository.NewSchedulePatientRepository(db)
	svc := NewStatsService(spRepo)
	ctx := context.Background()

	tr := &model.User{Email: "t@x.com", PasswordHash: "h", Name: "T", Role: "translator", Status: "active"}
	require.NoError(t, db.Create(tr).Error)
	p := &model.Patient{Name: "P", Phone: "1", IDType: "passport", IDNumber: "X1"}
	require.NoError(t, db.Create(p).Error)

	mk := func(date time.Time, actual int) {
		s := &model.Schedule{TranslatorID: tr.ID, Date: date, StartTime: "09:00", EndTime: "12:00", Location: "L"}
		require.NoError(t, db.Create(s).Error)
		require.NoError(t, spRepo.CreateBatch([]*model.SchedulePatient{
			{ScheduleID: s.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00", ActualAmount: actual},
		}))
	}
	now := time.Now()
	mk(now, 500)                       // this month → counted
	mk(now.AddDate(0, -2, 0), 999)     // two months ago → excluded

	label, total, err := svc.CurrentMonthActualTotal(ctx)
	require.NoError(t, err)
	assert.Equal(t, now.Format("2006-01"), label)
	assert.EqualValues(t, 500, total)
}

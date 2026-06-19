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

func newSchedulePatientTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Patient{}, &model.Schedule{},
		&model.SchedulePatient{}, &model.DiagnosisPhoto{},
	))
	return db
}

func seedPatient(t *testing.T, db *gorm.DB, name, idNumber string) *model.Patient {
	t.Helper()
	p := &model.Patient{Name: name, Phone: "1", IDType: "passport", IDNumber: idNumber}
	require.NoError(t, db.Create(p).Error)
	return p
}

func seedSchedule(t *testing.T, db *gorm.DB, translatorID uint) *model.Schedule {
	t.Helper()
	s := &model.Schedule{
		TranslatorID: translatorID,
		StartTime:    "09:00", EndTime: "12:00",
		Location: "L",
	}
	require.NoError(t, db.Create(s).Error)
	return s
}

func TestSchedulePatientRepo_CreateBatch(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p1 := seedPatient(t, db, "A", "ID1")
	p2 := seedPatient(t, db, "B", "ID2")

	repo := NewSchedulePatientRepository(db)
	err := repo.CreateBatch([]*model.SchedulePatient{
		{ScheduleID: sch.ID, PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00"},
		{ScheduleID: sch.ID, PatientID: p2.ID, StartTime: "10:00", EndTime: "11:00"},
	})
	require.NoError(t, err)

	got, err := repo.FindByScheduleID(sch.ID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestSchedulePatientRepo_FindByScheduleID_PreloadsPatient(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p := seedPatient(t, db, "Alice", "X1")
	repo := NewSchedulePatientRepository(db)
	require.NoError(t, repo.CreateBatch([]*model.SchedulePatient{
		{ScheduleID: sch.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00"},
	}))

	got, err := repo.FindByScheduleID(sch.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Alice", got[0].Patient.Name, "Patient relation should be preloaded")
}

func TestSchedulePatientRepo_DeleteByScheduleID(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p := seedPatient(t, db, "A", "X1")
	repo := NewSchedulePatientRepository(db)
	require.NoError(t, repo.CreateBatch([]*model.SchedulePatient{
		{ScheduleID: sch.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00"},
	}))

	require.NoError(t, repo.DeleteByScheduleID(sch.ID))

	got, err := repo.FindByScheduleID(sch.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSchedulePatientRepo_DeleteByScheduleIDs_Bulk(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	s1 := seedSchedule(t, db, 0)
	s2 := seedSchedule(t, db, 0)
	s3 := seedSchedule(t, db, 0) // 不刪
	p := seedPatient(t, db, "A", "X1")
	repo := NewSchedulePatientRepository(db)
	require.NoError(t, repo.CreateBatch([]*model.SchedulePatient{
		{ScheduleID: s1.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00"},
		{ScheduleID: s2.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00"},
		{ScheduleID: s3.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00"},
	}))

	require.NoError(t, repo.DeleteByScheduleIDs([]uint{s1.ID, s2.ID}))

	left, _ := repo.FindByScheduleID(s3.ID)
	assert.Len(t, left, 1, "s3 should be untouched")
	empty1, _ := repo.FindByScheduleID(s1.ID)
	assert.Empty(t, empty1)
	empty2, _ := repo.FindByScheduleID(s2.ID)
	assert.Empty(t, empty2)
}

func TestSchedulePatientRepo_UpdateStatus(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p := seedPatient(t, db, "A", "X1")
	repo := NewSchedulePatientRepository(db)
	sp := &model.SchedulePatient{ScheduleID: sch.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00"}
	require.NoError(t, repo.CreateBatch([]*model.SchedulePatient{sp}))

	require.NoError(t, repo.UpdateStatus(sp.ID, model.SchedulePatientStatusNoShow, "patient called to cancel"))

	got, err := repo.FindByID(sp.ID)
	require.NoError(t, err)
	assert.Equal(t, model.SchedulePatientStatusNoShow, got.Status)
	assert.Equal(t, "patient called to cancel", got.NoShowReason)
}

func TestSchedulePatientRepo_SumActualByPatients(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p1 := seedPatient(t, db, "A", "ID1")
	p2 := seedPatient(t, db, "B", "ID2")
	p3 := seedPatient(t, db, "C", "ID3") // 完全沒有看診紀錄 → map 中缺席

	repo := NewSchedulePatientRepository(db)
	require.NoError(t, repo.CreateBatch([]*model.SchedulePatient{
		{ScheduleID: sch.ID, PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00", ActualAmount: 300},
	}))
	// p1 在另一個 schedule 又有一筆 → 應加總；p2 的一筆為 no_show（actual=0）
	sch2 := seedSchedule(t, db, 0)
	require.NoError(t, repo.CreateBatch([]*model.SchedulePatient{
		{ScheduleID: sch2.ID, PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00", ActualAmount: 500},
		{ScheduleID: sch2.ID, PatientID: p2.ID, StartTime: "10:00", EndTime: "11:00", ActualAmount: 0},
	}))

	got, err := repo.SumActualByPatients([]uint{p1.ID, p2.ID, p3.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 800, got[p1.ID], "p1 = 300 + 500")
	assert.EqualValues(t, 0, got[p2.ID], "p2 only no_show (actual=0)")
	_, ok := got[p3.ID]
	assert.False(t, ok, "patient with no rows should be absent from the map")
}

func TestSchedulePatientRepo_SumActualByPatientDateRange(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	repo := NewSchedulePatientRepository(db)
	p := seedPatient(t, db, "A", "ID1")

	mk := func(date time.Time, actual int) {
		s := &model.Schedule{TranslatorID: 0, Date: date, StartTime: "09:00", EndTime: "12:00", Location: "L"}
		require.NoError(t, db.Create(s).Error)
		require.NoError(t, repo.CreateBatch([]*model.SchedulePatient{
			{ScheduleID: s.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00", ActualAmount: actual},
		}))
	}
	mk(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 100)   // 2026 lower edge → included
	mk(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), 200) // 2026 upper edge → included
	mk(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), 999) // 2025 → excluded
	mk(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), 999)   // 2027 → excluded (half-open upper)

	// Half-open [from, to): 2026-01-01 .. 2027-01-01.
	got, err := repo.SumActualByPatientDateRange(p.ID, "2026-01-01", "2027-01-01")
	require.NoError(t, err)
	assert.EqualValues(t, 300, got, "only 2026 visits (100+200)")
}

func TestSchedulePatientRepo_SumActualByPatients_EmptyInput(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	repo := NewSchedulePatientRepository(db)
	got, err := repo.SumActualByPatients(nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSchedulePatientRepo_UniqueConstraintEnforced(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p := seedPatient(t, db, "A", "X1")
	repo := NewSchedulePatientRepository(db)
	require.NoError(t, repo.CreateBatch([]*model.SchedulePatient{
		{ScheduleID: sch.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00"},
	}))

	// 同 schedule + 同 patient 第二次插入應失敗（unique index）
	err := repo.CreateBatch([]*model.SchedulePatient{
		{ScheduleID: sch.ID, PatientID: p.ID, StartTime: "11:00", EndTime: "12:00"},
	})
	assert.Error(t, err, "unique (scheduleID, patientID) should reject duplicates")
}

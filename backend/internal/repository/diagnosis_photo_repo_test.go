package repository

import (
	"testing"
	"time"

	"translator-checkin/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiagnosisPhotoRepo_CreateAndFindBySchedulePatient(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p := seedPatient(t, db, "A", "X1")

	spRepo := NewSchedulePatientRepository(db)
	sp := &model.SchedulePatient{ScheduleID: sch.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00"}
	require.NoError(t, spRepo.CreateBatch([]*model.SchedulePatient{sp}))

	repo := NewDiagnosisPhotoRepository(db)
	require.NoError(t, repo.Create(&model.DiagnosisPhoto{
		SchedulePatientID: sp.ID, PhotoURL: "/u/d1.jpg", UploadedAt: time.Now(),
	}))
	require.NoError(t, repo.Create(&model.DiagnosisPhoto{
		SchedulePatientID: sp.ID, PhotoURL: "/u/d2.jpg", UploadedAt: time.Now(),
	}))

	photos, err := repo.FindBySchedulePatientID(sp.ID)
	require.NoError(t, err)
	assert.Len(t, photos, 2)
}

func TestDiagnosisPhotoRepo_CountBySchedulePatient(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p := seedPatient(t, db, "A", "X1")

	spRepo := NewSchedulePatientRepository(db)
	sp := &model.SchedulePatient{ScheduleID: sch.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00"}
	require.NoError(t, spRepo.CreateBatch([]*model.SchedulePatient{sp}))

	repo := NewDiagnosisPhotoRepository(db)
	count, err := repo.CountBySchedulePatientID(sp.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 0, count)

	require.NoError(t, repo.Create(&model.DiagnosisPhoto{SchedulePatientID: sp.ID, PhotoURL: "/u/d.jpg", UploadedAt: time.Now()}))
	count, _ = repo.CountBySchedulePatientID(sp.ID)
	assert.EqualValues(t, 1, count)
}

func TestDiagnosisPhotoRepo_FindByIDAndDelete(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p := seedPatient(t, db, "A", "X1")

	spRepo := NewSchedulePatientRepository(db)
	sp := &model.SchedulePatient{ScheduleID: sch.ID, PatientID: p.ID, StartTime: "09:00", EndTime: "10:00"}
	require.NoError(t, spRepo.CreateBatch([]*model.SchedulePatient{sp}))

	repo := NewDiagnosisPhotoRepository(db)
	photo := &model.DiagnosisPhoto{SchedulePatientID: sp.ID, PhotoURL: "/u/d1.jpg", UploadedAt: time.Now()}
	require.NoError(t, repo.Create(photo))

	// FindByID returns the row.
	got, err := repo.FindByID(photo.ID)
	require.NoError(t, err)
	assert.Equal(t, "/u/d1.jpg", got.PhotoURL)
	assert.Equal(t, sp.ID, got.SchedulePatientID)

	// Delete removes it; subsequent FindByID errors.
	require.NoError(t, repo.Delete(photo.ID))
	_, err = repo.FindByID(photo.ID)
	assert.Error(t, err)

	count, _ := repo.CountBySchedulePatientID(sp.ID)
	assert.EqualValues(t, 0, count)
}

func TestDiagnosisPhotoRepo_FindBySchedulePatientIDs_Batched(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p1 := seedPatient(t, db, "A", "X1")
	p2 := seedPatient(t, db, "B", "X2")
	p3 := seedPatient(t, db, "C", "X3")

	spRepo := NewSchedulePatientRepository(db)
	sp1 := &model.SchedulePatient{ScheduleID: sch.ID, PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00"}
	sp2 := &model.SchedulePatient{ScheduleID: sch.ID, PatientID: p2.ID, StartTime: "10:00", EndTime: "11:00"}
	sp3 := &model.SchedulePatient{ScheduleID: sch.ID, PatientID: p3.ID, StartTime: "11:00", EndTime: "12:00"}
	require.NoError(t, spRepo.CreateBatch([]*model.SchedulePatient{sp1, sp2, sp3}))

	repo := NewDiagnosisPhotoRepository(db)
	require.NoError(t, repo.Create(&model.DiagnosisPhoto{SchedulePatientID: sp1.ID, PhotoURL: "/u/1a.jpg", UploadedAt: time.Now()}))
	require.NoError(t, repo.Create(&model.DiagnosisPhoto{SchedulePatientID: sp1.ID, PhotoURL: "/u/1b.jpg", UploadedAt: time.Now().Add(time.Second)}))
	require.NoError(t, repo.Create(&model.DiagnosisPhoto{SchedulePatientID: sp2.ID, PhotoURL: "/u/2.jpg", UploadedAt: time.Now()}))
	// sp3 has no photos.

	got, err := repo.FindBySchedulePatientIDs([]uint{sp1.ID, sp2.ID, sp3.ID})
	require.NoError(t, err)
	// All photos for the requested ids, in one query (sp1 ×2, sp2 ×1, sp3 ×0).
	assert.Len(t, got, 3)

	// Empty input → empty result, no query needed.
	none, err := repo.FindBySchedulePatientIDs(nil)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestDiagnosisPhotoRepo_FindBySchedulePatient_OtherSPIsolated(t *testing.T) {
	db := newSchedulePatientTestDB(t)
	sch := seedSchedule(t, db, 0)
	p1 := seedPatient(t, db, "A", "X1")
	p2 := seedPatient(t, db, "B", "X2")

	spRepo := NewSchedulePatientRepository(db)
	sp1 := &model.SchedulePatient{ScheduleID: sch.ID, PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00"}
	sp2 := &model.SchedulePatient{ScheduleID: sch.ID, PatientID: p2.ID, StartTime: "10:00", EndTime: "11:00"}
	require.NoError(t, spRepo.CreateBatch([]*model.SchedulePatient{sp1, sp2}))

	repo := NewDiagnosisPhotoRepository(db)
	require.NoError(t, repo.Create(&model.DiagnosisPhoto{SchedulePatientID: sp1.ID, PhotoURL: "/u/1.jpg", UploadedAt: time.Now()}))
	require.NoError(t, repo.Create(&model.DiagnosisPhoto{SchedulePatientID: sp2.ID, PhotoURL: "/u/2.jpg", UploadedAt: time.Now()}))

	got, _ := repo.FindBySchedulePatientID(sp1.ID)
	assert.Len(t, got, 1)
	assert.Equal(t, "/u/1.jpg", got[0].PhotoURL)
}

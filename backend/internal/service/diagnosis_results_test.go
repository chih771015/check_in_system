package service

import (
	"context"
	"testing"
	"time"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resultsFixture struct {
	diagSvc      *DiagnosisService
	scheduleSvc  *ScheduleService
	patientRepo  *repository.PatientRepository
	scheduleRepo *repository.ScheduleRepository
	spRepo       *repository.SchedulePatientRepository
	photoRepo    *repository.DiagnosisPhotoRepository
	userRepo     *repository.UserRepository
	tr           *model.User
	other        *model.User
}

func newResultsFixture(t *testing.T) *resultsFixture {
	t.Helper()
	db := newTestDB(t)
	scheduleRepo := repository.NewScheduleRepository(db)
	checkinRepo := repository.NewCheckinRepository(db)
	userRepo := repository.NewUserRepository(db)
	patientRepo := repository.NewPatientRepository(db)
	spRepo := repository.NewSchedulePatientRepository(db)
	photoRepo := repository.NewDiagnosisPhotoRepository(db)

	scheduleSvc := NewScheduleService(scheduleRepo, checkinRepo, userRepo).
		WithPatientRepos(spRepo, patientRepo)

	tr := &model.User{Email: "t@x.com", PasswordHash: "h", Name: "Alice", Role: "translator", Status: "active"}
	other := &model.User{Email: "o@x.com", PasswordHash: "h", Name: "Bob", Role: "translator", Status: "active"}
	require.NoError(t, userRepo.Create(tr))
	require.NoError(t, userRepo.Create(other))

	return &resultsFixture{
		diagSvc:      NewDiagnosisService(spRepo, photoRepo, scheduleRepo),
		scheduleSvc:  scheduleSvc,
		patientRepo:  patientRepo,
		scheduleRepo: scheduleRepo,
		spRepo:       spRepo,
		photoRepo:    photoRepo,
		userRepo:     userRepo,
		tr:           tr,
		other:        other,
	}
}

func (fx *resultsFixture) seedPatient(t *testing.T, name, idNum string) *model.Patient {
	t.Helper()
	p := &model.Patient{Name: name, Phone: "0900" + idNum, IDType: "passport", IDNumber: idNum}
	require.NoError(t, fx.patientRepo.Create(p))
	return p
}

// seedSchedule creates a schedule for tr on the given date with one patient
// and returns the SchedulePatient id so callers can flip its status.
func (fx *resultsFixture) seedSchedule(t *testing.T, tr *model.User, date, start, end string, patient *model.Patient) uint {
	t.Helper()
	resp, err := fx.scheduleSvc.Create(context.Background(), dto.CreateScheduleRequest{
		TranslatorID: tr.ID, Date: date, StartTime: "09:00", EndTime: "17:00", Location: "L",
		Patients: []dto.SchedulePatientPayload{
			{PatientID: patient.ID, StartTime: start, EndTime: end},
		},
	})
	require.NoError(t, err)
	sps, _ := fx.spRepo.FindByScheduleID(resp.ID)
	require.Len(t, sps, 1)
	return sps[0].ID
}

func TestDiagnosisService_ListResults_ExcludesPending(t *testing.T) {
	fx := newResultsFixture(t)
	p1 := fx.seedPatient(t, "P1", "1")
	p2 := fx.seedPatient(t, "P2", "2")
	p3 := fx.seedPatient(t, "P3", "3")
	sp1 := fx.seedSchedule(t, fx.tr, "2026-06-01", "09:00", "10:00", p1)
	fx.seedSchedule(t, fx.tr, "2026-06-02", "09:00", "10:00", p2) // stays pending
	sp3 := fx.seedSchedule(t, fx.tr, "2026-06-03", "09:00", "10:00", p3)

	ctx := context.Background()
	require.NoError(t, fx.diagSvc.UploadDiagnosis(ctx, fx.tr.ID, sp1, []string{"/u/1.jpg"}))
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp3, "patient cancelled"))

	resp, err := fx.diagSvc.ListResults(ctx, dto.DiagnosisResultsQuery{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, resp.Total, "pending row should not appear")
	require.Len(t, resp.Data, 2)
	for _, r := range resp.Data {
		assert.NotEqual(t, "pending", r.Status)
	}
}

func TestDiagnosisService_ListResults_SortedByDateAndTimeDesc(t *testing.T) {
	fx := newResultsFixture(t)
	p1 := fx.seedPatient(t, "P1", "1")
	p2 := fx.seedPatient(t, "P2", "2")
	p3 := fx.seedPatient(t, "P3", "3")
	// Three on same date with different times, plus one earlier date
	sp1 := fx.seedSchedule(t, fx.tr, "2026-06-01", "09:00", "10:00", p1)
	sp2 := fx.seedSchedule(t, fx.tr, "2026-06-02", "09:00", "10:00", p2)
	sp3 := fx.seedSchedule(t, fx.tr, "2026-06-02", "14:00", "15:00", p3)

	ctx := context.Background()
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp1, "r1"))
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp2, "r2"))
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp3, "r3"))

	resp, err := fx.diagSvc.ListResults(ctx, dto.DiagnosisResultsQuery{})
	require.NoError(t, err)
	require.Len(t, resp.Data, 3)
	// Expect: 06-02 14:00, 06-02 09:00, 06-01 09:00
	assert.Equal(t, "2026-06-02", resp.Data[0].Date)
	assert.Equal(t, "14:00", resp.Data[0].StartTime)
	assert.Equal(t, "2026-06-02", resp.Data[1].Date)
	assert.Equal(t, "09:00", resp.Data[1].StartTime)
	assert.Equal(t, "2026-06-01", resp.Data[2].Date)
}

func TestDiagnosisService_ListResults_FilterByDateRange(t *testing.T) {
	fx := newResultsFixture(t)
	p1 := fx.seedPatient(t, "P1", "1")
	p2 := fx.seedPatient(t, "P2", "2")
	p3 := fx.seedPatient(t, "P3", "3")
	sp1 := fx.seedSchedule(t, fx.tr, "2026-06-01", "09:00", "10:00", p1)
	sp2 := fx.seedSchedule(t, fx.tr, "2026-06-05", "09:00", "10:00", p2)
	sp3 := fx.seedSchedule(t, fx.tr, "2026-06-10", "09:00", "10:00", p3)

	ctx := context.Background()
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp1, "r1"))
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp2, "r2"))
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp3, "r3"))

	// 06-02 to 06-08 → only the middle one
	resp, err := fx.diagSvc.ListResults(ctx, dto.DiagnosisResultsQuery{DateFrom: "2026-06-02", DateTo: "2026-06-08"})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "2026-06-05", resp.Data[0].Date)
}

func TestDiagnosisService_ListResults_FilterByTranslator(t *testing.T) {
	fx := newResultsFixture(t)
	p1 := fx.seedPatient(t, "P1", "1")
	p2 := fx.seedPatient(t, "P2", "2")
	sp1 := fx.seedSchedule(t, fx.tr, "2026-06-01", "09:00", "10:00", p1)
	sp2 := fx.seedSchedule(t, fx.other, "2026-06-02", "09:00", "10:00", p2)

	ctx := context.Background()
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp1, "r1"))
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.other.ID, sp2, "r2"))

	resp, err := fx.diagSvc.ListResults(ctx, dto.DiagnosisResultsQuery{TranslatorID: fx.other.ID})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, fx.other.Name, resp.Data[0].TranslatorName)
}

func TestDiagnosisService_ListResults_FilterByStatus(t *testing.T) {
	fx := newResultsFixture(t)
	p1 := fx.seedPatient(t, "P1", "1")
	p2 := fx.seedPatient(t, "P2", "2")
	sp1 := fx.seedSchedule(t, fx.tr, "2026-06-01", "09:00", "10:00", p1)
	sp2 := fx.seedSchedule(t, fx.tr, "2026-06-02", "09:00", "10:00", p2)

	ctx := context.Background()
	require.NoError(t, fx.diagSvc.UploadDiagnosis(ctx, fx.tr.ID, sp1, []string{"/u/1.jpg"}))
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp2, "cancelled"))

	completedOnly, err := fx.diagSvc.ListResults(context.Background(), dto.DiagnosisResultsQuery{Status: "completed"})
	require.NoError(t, err)
	require.Len(t, completedOnly.Data, 1)
	assert.Equal(t, "completed", completedOnly.Data[0].Status)

	noShowOnly, err := fx.diagSvc.ListResults(context.Background(), dto.DiagnosisResultsQuery{Status: "no_show"})
	require.NoError(t, err)
	require.Len(t, noShowOnly.Data, 1)
	assert.Equal(t, "no_show", noShowOnly.Data[0].Status)
}

func TestDiagnosisService_ListResults_FilterByPatientName(t *testing.T) {
	fx := newResultsFixture(t)
	p1 := fx.seedPatient(t, "John Doe", "1")
	p2 := fx.seedPatient(t, "Jane Smith", "2")
	sp1 := fx.seedSchedule(t, fx.tr, "2026-06-01", "09:00", "10:00", p1)
	sp2 := fx.seedSchedule(t, fx.tr, "2026-06-02", "09:00", "10:00", p2)

	ctx := context.Background()
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp1, "r1"))
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sp2, "r2"))

	resp, err := fx.diagSvc.ListResults(ctx, dto.DiagnosisResultsQuery{PatientName: "John"})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "John Doe", resp.Data[0].PatientName)
}

func TestDiagnosisService_ListResults_IncludesPhotosAndPatientFields(t *testing.T) {
	fx := newResultsFixture(t)
	p := fx.seedPatient(t, "Alice Patient", "ABC123")
	sp := fx.seedSchedule(t, fx.tr, "2026-06-01", "09:00", "10:00", p)

	ctx := context.Background()
	require.NoError(t, fx.diagSvc.UploadDiagnosis(ctx, fx.tr.ID, sp, []string{"/u/a.jpg", "/u/b.jpg"}))

	resp, err := fx.diagSvc.ListResults(ctx, dto.DiagnosisResultsQuery{})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	r := resp.Data[0]
	assert.Equal(t, "Alice Patient", r.PatientName)
	assert.Equal(t, "0900ABC123", r.PatientPhone)
	assert.Equal(t, "passport", r.IDType)
	assert.Equal(t, "ABC123", r.IDNumber)
	assert.Len(t, r.DiagnosisPhotos, 2)
	// No-show reason should be empty for completed
	assert.Empty(t, r.NoShowReason)
	// Translator
	assert.Equal(t, fx.tr.Name, r.TranslatorName)
}

func TestDiagnosisService_ListResults_Pagination(t *testing.T) {
	fx := newResultsFixture(t)
	// Seed 5 results
	for i := 0; i < 5; i++ {
		p := fx.seedPatient(t, "P", string(rune('A'+i)))
		sp := fx.seedSchedule(t, fx.tr, "2026-06-0"+string(rune('1'+i)), "09:00", "10:00", p)
		require.NoError(t, fx.diagSvc.MarkNoShow(context.Background(), fx.tr.ID, sp, "r"))
	}
	resp, err := fx.diagSvc.ListResults(context.Background(), dto.DiagnosisResultsQuery{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.EqualValues(t, 5, resp.Total)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 2, resp.PageSize)
}

func TestDiagnosisService_ListResults_UpdatedAtPresent(t *testing.T) {
	fx := newResultsFixture(t)
	p := fx.seedPatient(t, "P", "1")
	sp := fx.seedSchedule(t, fx.tr, "2026-06-01", "09:00", "10:00", p)
	before := time.Now().Add(-time.Second)
	require.NoError(t, fx.diagSvc.MarkNoShow(context.Background(), fx.tr.ID, sp, "r1"))

	resp, err := fx.diagSvc.ListResults(context.Background(), dto.DiagnosisResultsQuery{})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.True(t, resp.Data[0].UpdatedAt.After(before),
		"updatedAt should reflect the no-show timestamp; got %v", resp.Data[0].UpdatedAt)
}

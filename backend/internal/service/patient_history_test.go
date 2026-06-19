package service

import (
	"context"
	"testing"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type historyFixture struct {
	patientSvc   *PatientService
	scheduleSvc  *ScheduleService
	diagSvc      *DiagnosisService
	patientRepo  *repository.PatientRepository
	scheduleRepo *repository.ScheduleRepository
	userRepo     *repository.UserRepository
	spRepo       *repository.SchedulePatientRepository
	photoRepo    *repository.DiagnosisPhotoRepository
	tr           *model.User
	patient      *model.Patient
}

func newHistoryFixture(t *testing.T) *historyFixture {
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
	patientSvc := NewPatientService(patientRepo).
		WithScopeRepo(spRepo).
		WithHistoryRepos(scheduleRepo, spRepo, photoRepo)
	diagSvc := NewDiagnosisService(spRepo, photoRepo, scheduleRepo)

	tr := &model.User{Email: "t@x.com", PasswordHash: "h", Name: "Alice", Role: "translator", Status: "active"}
	require.NoError(t, userRepo.Create(tr))
	patient := &model.Patient{Name: "P", Phone: "1", IDType: "passport", IDNumber: "X1"}
	require.NoError(t, patientRepo.Create(patient))

	return &historyFixture{
		patientSvc:   patientSvc,
		scheduleSvc:  scheduleSvc,
		diagSvc:      diagSvc,
		patientRepo:  patientRepo,
		scheduleRepo: scheduleRepo,
		userRepo:     userRepo,
		spRepo:       spRepo,
		photoRepo:    photoRepo,
		tr:           tr,
		patient:      patient,
	}
}

func TestPatientService_GetHistory_AggregatesVisits(t *testing.T) {
	fx := newHistoryFixture(t)
	ctx := context.Background()

	// Schedule 1: completed with 2 photos
	resp1, err := fx.scheduleSvc.Create(ctx, dto.CreateScheduleRequest{
		TranslatorID: fx.tr.ID, Date: "2026-06-01", StartTime: "09:00", EndTime: "12:00", Location: "Hospital A",
		Patients: []dto.SchedulePatientPayload{
			{PatientID: fx.patient.ID, StartTime: "09:00", EndTime: "10:00"},
		},
	})
	require.NoError(t, err)
	sps1, _ := fx.spRepo.FindByScheduleID(resp1.ID)
	require.NoError(t, fx.diagSvc.UploadDiagnosis(ctx, fx.tr.ID, sps1[0].ID, []string{"/u/p1.jpg", "/u/p2.jpg"}))

	// Schedule 2: no_show
	resp2, err := fx.scheduleSvc.Create(ctx, dto.CreateScheduleRequest{
		TranslatorID: fx.tr.ID, Date: "2026-06-05", StartTime: "14:00", EndTime: "17:00", Location: "Hospital B",
		Patients: []dto.SchedulePatientPayload{
			{PatientID: fx.patient.ID, StartTime: "14:00", EndTime: "15:00"},
		},
	})
	require.NoError(t, err)
	sps2, _ := fx.spRepo.FindByScheduleID(resp2.ID)
	require.NoError(t, fx.diagSvc.MarkNoShow(ctx, fx.tr.ID, sps2[0].ID, "patient called to cancel"))

	hist, err := fx.patientSvc.GetHistory(ctx, fx.patient.ID, "", "")
	require.NoError(t, err)
	require.Len(t, hist.History, 2, "should return 2 history entries")

	// Order: DESC by date — Hospital B first (2026-06-05), then Hospital A (2026-06-01)
	first := hist.History[0]
	assert.Equal(t, "2026-06-05", first.Date)
	assert.Equal(t, "Hospital B", first.Location)
	assert.Equal(t, "Alice", first.TranslatorName)
	assert.Equal(t, "no_show", first.Status)
	assert.Equal(t, "patient called to cancel", first.NoShowReason)
	assert.Empty(t, first.DiagnosisPhotos)

	second := hist.History[1]
	assert.Equal(t, "2026-06-01", second.Date)
	assert.Equal(t, "Hospital A", second.Location)
	assert.Equal(t, "completed", second.Status)
	assert.Len(t, second.DiagnosisPhotos, 2)
}

func TestPatientService_GetHistory_Empty(t *testing.T) {
	fx := newHistoryFixture(t)
	hist, err := fx.patientSvc.GetHistory(context.Background(), fx.patient.ID, "", "")
	require.NoError(t, err)
	assert.Empty(t, hist.History)
	assert.EqualValues(t, 0, hist.ActualTotal)
}

func TestPatientService_GetHistory_DateRangeAndTotal(t *testing.T) {
	fx := newHistoryFixture(t)
	ctx := context.Background()

	// Two visits with actual amounts on different months.
	mk := func(date string, actual int) {
		resp, err := fx.scheduleSvc.Create(ctx, dto.CreateScheduleRequest{
			TranslatorID: fx.tr.ID, Date: date, StartTime: "09:00", EndTime: "12:00", Location: "L",
			Patients: []dto.SchedulePatientPayload{
				{PatientID: fx.patient.ID, StartTime: "09:00", EndTime: "10:00"},
			},
		})
		require.NoError(t, err)
		sps, _ := fx.spRepo.FindByScheduleID(resp.ID)
		require.NoError(t, fx.spRepo.UpdateActualAmount(sps[0].ID, actual))
	}
	mk("2026-05-10", 300)
	mk("2026-06-20", 500)

	// No range → all entries + all-time total.
	all, err := fx.patientSvc.GetHistory(ctx, fx.patient.ID, "", "")
	require.NoError(t, err)
	assert.Len(t, all.History, 2)
	assert.EqualValues(t, 800, all.ActualTotal)

	// Range covering only May → 1 entry + range total.
	may, err := fx.patientSvc.GetHistory(ctx, fx.patient.ID, "2026-05-01", "2026-05-31")
	require.NoError(t, err)
	require.Len(t, may.History, 1)
	assert.Equal(t, "2026-05-10", may.History[0].Date)
	assert.EqualValues(t, 300, may.ActualTotal)

	// Inclusive upper bound: range ending exactly on the visit date includes it.
	junEdge, err := fx.patientSvc.GetHistory(ctx, fx.patient.ID, "2026-06-20", "2026-06-20")
	require.NoError(t, err)
	require.Len(t, junEdge.History, 1)
	assert.EqualValues(t, 500, junEdge.ActualTotal)
}

func TestPatientService_ActualTotals(t *testing.T) {
	fx := newHistoryFixture(t)
	ctx := context.Background()
	p2 := &model.Patient{Name: "P2", Phone: "1", IDType: "passport", IDNumber: "X2"}
	require.NoError(t, fx.patientRepo.Create(p2))

	resp, err := fx.scheduleSvc.Create(ctx, dto.CreateScheduleRequest{
		TranslatorID: fx.tr.ID, Date: "2026-05-10", StartTime: "09:00", EndTime: "12:00", Location: "L",
		Patients: []dto.SchedulePatientPayload{
			{PatientID: fx.patient.ID, StartTime: "09:00", EndTime: "10:00"},
		},
	})
	require.NoError(t, err)
	sps, _ := fx.spRepo.FindByScheduleID(resp.ID)
	require.NoError(t, fx.spRepo.UpdateActualAmount(sps[0].ID, 700))

	totals, err := fx.patientSvc.ActualTotals(ctx, []uint{fx.patient.ID, p2.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 700, totals[fx.patient.ID])
	_, ok := totals[p2.ID]
	assert.False(t, ok, "patient with no visits absent from map")
}

func TestPatientService_GetHistory_PatientNotFound(t *testing.T) {
	fx := newHistoryFixture(t)
	_, err := fx.patientSvc.GetHistory(context.Background(), 99999, "", "")
	assert.Error(t, err)
}

package service

import (
	"context"
	"errors"
	"testing"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type diagFixture struct {
	svc          *DiagnosisService
	spRepo       *repository.SchedulePatientRepository
	scheduleRepo *repository.ScheduleRepository
	photoRepo    *repository.DiagnosisPhotoRepository
	scheduleSvc  *ScheduleService
	patientRepo  *repository.PatientRepository
	userRepo     *repository.UserRepository
	translator   *model.User
	other        *model.User
	schedule     *model.Schedule
	sp           *model.SchedulePatient
}

func newDiagFixture(t *testing.T) *diagFixture {
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

	tr := &model.User{Email: "t@x.com", PasswordHash: "h", Name: "T", Role: "translator", Status: "active"}
	other := &model.User{Email: "o@x.com", PasswordHash: "h", Name: "O", Role: "translator", Status: "active"}
	require.NoError(t, userRepo.Create(tr))
	require.NoError(t, userRepo.Create(other))

	patient := &model.Patient{Name: "P", Phone: "1", IDType: "passport", IDNumber: "X1"}
	require.NoError(t, patientRepo.Create(patient))

	resp, err := scheduleSvc.Create(context.Background(), dto.CreateScheduleRequest{
		TranslatorID: tr.ID, Date: "2026-06-01", StartTime: "09:00", EndTime: "12:00", Location: "L",
		Patients: []dto.SchedulePatientPayload{
			{PatientID: patient.ID, StartTime: "09:00", EndTime: "10:00"},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Patients, 1)

	// Fetch the SchedulePatient row by querying via spRepo
	rows, _ := spRepo.FindByScheduleID(resp.ID)
	require.Len(t, rows, 1)

	sched, err := scheduleRepo.FindByID(resp.ID)
	require.NoError(t, err)

	return &diagFixture{
		svc:          NewDiagnosisService(spRepo, photoRepo, scheduleRepo),
		spRepo:       spRepo,
		scheduleRepo: scheduleRepo,
		photoRepo:    photoRepo,
		scheduleSvc:  scheduleSvc,
		patientRepo:  patientRepo,
		userRepo:     userRepo,
		translator:   tr,
		other:        other,
		schedule:     sched,
		sp:           &rows[0],
	}
}

func TestDiagnosisService_UploadDiagnosis_Success(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()

	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/1.jpg", "/u/2.jpg"}))

	photos, _ := fx.photoRepo.FindBySchedulePatientID(fx.sp.ID)
	assert.Len(t, photos, 2)

	reloaded, _ := fx.spRepo.FindByID(fx.sp.ID)
	assert.Equal(t, model.SchedulePatientStatusCompleted, reloaded.Status)
}

func TestDiagnosisService_UploadDiagnosis_ExceedsLimit(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	// 1 + 3 → exceeds limit (3)
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/1.jpg"}))
	err := fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/2.jpg", "/u/3.jpg", "/u/4.jpg"})
	assert.True(t, errors.Is(err, ErrDiagnosisPhotoLimit))
}

func TestDiagnosisService_UploadDiagnosis_NotOwned(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	err := fx.svc.UploadDiagnosis(ctx, fx.other.ID, fx.sp.ID, []string{"/u/1.jpg"})
	assert.True(t, errors.Is(err, ErrDiagnosisNotOwned))
}

func TestDiagnosisService_UploadDiagnosis_SchedulePatientNotFound(t *testing.T) {
	fx := newDiagFixture(t)
	err := fx.svc.UploadDiagnosis(context.Background(), fx.translator.ID, 99999, []string{"/u/1.jpg"})
	assert.True(t, errors.Is(err, ErrSchedulePatientNotFound))
}

func TestDiagnosisService_MarkNoShow_Success(t *testing.T) {
	fx := newDiagFixture(t)
	require.NoError(t, fx.svc.MarkNoShow(context.Background(), fx.translator.ID, fx.sp.ID, "patient called to cancel"))

	reloaded, _ := fx.spRepo.FindByID(fx.sp.ID)
	assert.Equal(t, model.SchedulePatientStatusNoShow, reloaded.Status)
	assert.Equal(t, "patient called to cancel", reloaded.NoShowReason)
}

func TestDiagnosisService_MarkNoShow_RequiresReason(t *testing.T) {
	fx := newDiagFixture(t)
	err := fx.svc.MarkNoShow(context.Background(), fx.translator.ID, fx.sp.ID, "")
	assert.True(t, errors.Is(err, ErrNoShowReasonRequired))
}

func TestDiagnosisService_MarkNoShow_NotOwned(t *testing.T) {
	fx := newDiagFixture(t)
	err := fx.svc.MarkNoShow(context.Background(), fx.other.ID, fx.sp.ID, "reason")
	assert.True(t, errors.Is(err, ErrDiagnosisNotOwned))
}

func TestDiagnosisService_AdminUploadDiagnosis_NoOwnerCheck(t *testing.T) {
	fx := newDiagFixture(t)
	// admin path bypasses translator ownership
	err := fx.svc.AdminUploadDiagnosis(context.Background(), fx.sp.ID, []string{"/u/a.jpg"})
	require.NoError(t, err)
	photos, _ := fx.photoRepo.FindBySchedulePatientID(fx.sp.ID)
	assert.Len(t, photos, 1)
}

func TestDiagnosisService_AdminMarkNoShow_Success(t *testing.T) {
	fx := newDiagFixture(t)
	err := fx.svc.AdminMarkNoShow(context.Background(), fx.sp.ID, "admin marked")
	require.NoError(t, err)
	reloaded, _ := fx.spRepo.FindByID(fx.sp.ID)
	assert.Equal(t, model.SchedulePatientStatusNoShow, reloaded.Status)
}

// ─── Phase 4.4 CheckOut gating ──────────────────────────────────────────────

func TestCheckinService_Leave_BlockedByPendingPatients(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()

	// Hook spRepo into the checkin service to enable the gate.
	checkinRepo := repository.NewCheckinRepository(fx.scheduleRepo.DB())
	checkinSvc := NewCheckinService(checkinRepo, fx.scheduleRepo, fx.userRepo, nil).
		WithSchedulePatientRepo(fx.spRepo)

	// First do arrive so leave isn't ErrArriveBeforeLeave
	_, err := checkinSvc.Checkin(ctx, fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "", false, "")
	require.NoError(t, err)

	// Pending patient → leave should be blocked
	_, err = checkinSvc.Checkin(ctx, fx.translator.ID, fx.schedule.ID, "leave",
		25.0, 121.5, "x", "/u/s2", "", false, "")
	assert.True(t, errors.Is(err, ErrCheckoutBlockedByPending),
		"expected ErrCheckoutBlockedByPending, got %v", err)
}

func TestCheckinService_Leave_PassesWhenAllPatientsProcessed(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()

	checkinRepo := repository.NewCheckinRepository(fx.scheduleRepo.DB())
	checkinSvc := NewCheckinService(checkinRepo, fx.scheduleRepo, fx.userRepo, nil).
		WithSchedulePatientRepo(fx.spRepo)

	_, err := checkinSvc.Checkin(ctx, fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "", false, "")
	require.NoError(t, err)

	// Mark patient as completed via diagnosis upload
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/d.jpg"}))

	// Leave should now succeed
	_, err = checkinSvc.Checkin(ctx, fx.translator.ID, fx.schedule.ID, "leave",
		25.0, 121.5, "x", "/u/s2", "", false, "")
	assert.NoError(t, err)
}

func TestCheckinService_Leave_MakeupBypassesGate(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()

	checkinRepo := repository.NewCheckinRepository(fx.scheduleRepo.DB())
	checkinSvc := NewCheckinService(checkinRepo, fx.scheduleRepo, fx.userRepo, nil).
		WithSchedulePatientRepo(fx.spRepo)

	_, err := checkinSvc.Checkin(ctx, fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "", true, "test makeup")
	require.NoError(t, err)

	// Even with pending patient, makeup leave should succeed (per stage-3 plan B).
	_, err = checkinSvc.Checkin(ctx, fx.translator.ID, fx.schedule.ID, "leave",
		25.0, 121.5, "x", "/u/s2", "", true, "test makeup leave")
	assert.NoError(t, err)
}

package service

import (
	"context"
	"errors"
	"testing"
	"time"

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
	checkinRepo  *repository.CheckinRepository
	scheduleSvc  *ScheduleService
	patientRepo  *repository.PatientRepository
	userRepo     *repository.UserRepository
	translator   *model.User
	other        *model.User
	schedule     *model.Schedule
	sp           *model.SchedulePatient
}

// seedLeaveCheckin records a "leave" check-in for the fixture's schedule so the
// post-leave diagnosis lock activates.
func (fx *diagFixture) seedLeaveCheckin(t *testing.T) {
	t.Helper()
	require.NoError(t, fx.checkinRepo.Create(&model.Checkin{
		ScheduleID:   fx.schedule.ID,
		TranslatorID: fx.translator.ID,
		Type:         "leave",
		CheckinTime:  time.Now(),
		SelfieURL:    "/uploads/leave.jpg",
	}))
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
		svc:          NewDiagnosisService(spRepo, photoRepo, scheduleRepo).WithCheckinRepo(checkinRepo),
		spRepo:       spRepo,
		scheduleRepo: scheduleRepo,
		photoRepo:    photoRepo,
		checkinRepo:  checkinRepo,
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

// ─── Diagnosis photo manage (list with IDs / delete / re-add) ───────────────

func TestDiagnosisService_ListPhotoItems_OwnerSeesIDs(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/1.jpg", "/u/2.jpg"}))

	items, err := fx.svc.ListPhotoItems(ctx, fx.translator.ID, fx.sp.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.NotZero(t, items[0].ID)
	assert.Equal(t, "/u/1.jpg", items[0].PhotoURL)
}

func TestDiagnosisService_ListPhotoItems_NotOwned(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/1.jpg"}))
	_, err := fx.svc.ListPhotoItems(ctx, fx.other.ID, fx.sp.ID)
	assert.True(t, errors.Is(err, ErrDiagnosisNotOwned))
}

func TestDiagnosisService_DeletePhoto_OwnerSuccess_KeepsCompletedWhenOthersRemain(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/1.jpg", "/u/2.jpg"}))
	items, _ := fx.svc.ListPhotoItems(ctx, fx.translator.ID, fx.sp.ID)

	require.NoError(t, fx.svc.DeletePhoto(ctx, fx.translator.ID, items[0].ID))

	remaining, _ := fx.photoRepo.FindBySchedulePatientID(fx.sp.ID)
	assert.Len(t, remaining, 1)
	// Still has a photo → stays completed.
	reloaded, _ := fx.spRepo.FindByID(fx.sp.ID)
	assert.Equal(t, model.SchedulePatientStatusCompleted, reloaded.Status)
}

func TestDiagnosisService_DeletePhoto_RevertsToPendingWhenLastRemoved(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/only.jpg"}))
	items, _ := fx.svc.ListPhotoItems(ctx, fx.translator.ID, fx.sp.ID)

	require.NoError(t, fx.svc.DeletePhoto(ctx, fx.translator.ID, items[0].ID))

	remaining, _ := fx.photoRepo.FindBySchedulePatientID(fx.sp.ID)
	assert.Len(t, remaining, 0)
	// No photos left → status reverts to pending so the slot is actionable again.
	reloaded, _ := fx.spRepo.FindByID(fx.sp.ID)
	assert.Equal(t, model.SchedulePatientStatusPending, reloaded.Status)
}

func TestDiagnosisService_DeletePhoto_NotOwned(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/1.jpg"}))
	items, _ := fx.svc.ListPhotoItems(ctx, fx.translator.ID, fx.sp.ID)

	err := fx.svc.DeletePhoto(ctx, fx.other.ID, items[0].ID)
	assert.True(t, errors.Is(err, ErrDiagnosisNotOwned))
}

func TestDiagnosisService_DeletePhoto_NotFound(t *testing.T) {
	fx := newDiagFixture(t)
	err := fx.svc.DeletePhoto(context.Background(), fx.translator.ID, 99999)
	assert.True(t, errors.Is(err, ErrDiagnosisPhotoNotFound))
}

func TestDiagnosisService_AdminDeletePhoto_NoOwnerCheck(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/1.jpg"}))
	items, _ := fx.svc.ListPhotoItems(ctx, fx.translator.ID, fx.sp.ID)

	require.NoError(t, fx.svc.AdminDeletePhoto(ctx, items[0].ID))
	remaining, _ := fx.photoRepo.FindBySchedulePatientID(fx.sp.ID)
	assert.Len(t, remaining, 0)
}

// ─── no_show 清空殘留照片（按錯更正）────────────────────────────────────────

func TestDiagnosisService_MarkNoShow_PurgesExistingPhotos(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/uploads/a.jpg", "/uploads/b.jpg"}))

	require.NoError(t, fx.svc.MarkNoShow(ctx, fx.translator.ID, fx.sp.ID, "patient no show"))

	// no_show 後照片應被清空，狀態與原因正確。
	photos, _ := fx.photoRepo.FindBySchedulePatientID(fx.sp.ID)
	assert.Len(t, photos, 0)
	reloaded, _ := fx.spRepo.FindByID(fx.sp.ID)
	assert.Equal(t, model.SchedulePatientStatusNoShow, reloaded.Status)
	assert.Equal(t, "patient no show", reloaded.NoShowReason)
}

func TestDiagnosisService_AdminMarkNoShow_PurgesExistingPhotos(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.AdminUploadDiagnosis(ctx, fx.sp.ID, []string{"/uploads/a.jpg"}))

	require.NoError(t, fx.svc.AdminMarkNoShow(ctx, fx.sp.ID, "admin no show"))

	photos, _ := fx.photoRepo.FindBySchedulePatientID(fx.sp.ID)
	assert.Len(t, photos, 0)
	reloaded, _ := fx.spRepo.FindByID(fx.sp.ID)
	assert.Equal(t, model.SchedulePatientStatusNoShow, reloaded.Status)
}

// ─── 離開打卡後鎖定翻譯員的診斷修改（管理員例外）──────────────────────────────

func TestDiagnosisService_UploadDiagnosis_AllowedAfterLeave(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	fx.seedLeaveCheckin(t)
	// X-ray / lab results may surface after departure → translator can still
	// ADD photos (only delete / no_show stay locked after leave).
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/uploads/a.jpg"}))
	photos, _ := fx.photoRepo.FindBySchedulePatientID(fx.sp.ID)
	assert.Len(t, photos, 1)
}

func TestDiagnosisService_DeletePhoto_LockedAfterLeave(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/uploads/a.jpg"}))
	items, _ := fx.svc.ListPhotoItems(ctx, fx.translator.ID, fx.sp.ID)
	fx.seedLeaveCheckin(t)

	err := fx.svc.DeletePhoto(ctx, fx.translator.ID, items[0].ID)
	assert.True(t, errors.Is(err, ErrDiagnosisLockedAfterLeave))
}

func TestDiagnosisService_MarkNoShow_LockedAfterLeave(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	fx.seedLeaveCheckin(t)
	err := fx.svc.MarkNoShow(ctx, fx.translator.ID, fx.sp.ID, "reason")
	assert.True(t, errors.Is(err, ErrDiagnosisLockedAfterLeave))
}

func TestDiagnosisService_ListPhotoItems_AllowedAfterLeave(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/uploads/a.jpg"}))
	fx.seedLeaveCheckin(t)
	// 唯讀仍允許（只是不能改）。
	_, err := fx.svc.ListPhotoItems(ctx, fx.translator.ID, fx.sp.ID)
	require.NoError(t, err)
}

func TestDiagnosisService_AdminBypassesLockAfterLeave(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	fx.seedLeaveCheckin(t)
	// 管理員不受離開鎖定限制。
	require.NoError(t, fx.svc.AdminUploadDiagnosis(ctx, fx.sp.ID, []string{"/uploads/a.jpg"}))
	items, _ := fx.svc.AdminListPhotoItems(ctx, fx.sp.ID)
	require.Len(t, items, 1)
	require.NoError(t, fx.svc.AdminDeletePhoto(ctx, items[0].ID))
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

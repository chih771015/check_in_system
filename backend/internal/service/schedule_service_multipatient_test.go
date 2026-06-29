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

// scheduleMultiFixture extends scheduleFixture with patient + schedule_patient
// repos so multi-patient tests can verify both sides.
type scheduleMultiFixture struct {
	*scheduleFixture
	patientRepo *repository.PatientRepository
	spRepo      *repository.SchedulePatientRepository
}

func newScheduleMultiFixture(t *testing.T) *scheduleMultiFixture {
	t.Helper()
	base := newScheduleFixture(t)
	db := base.scheduleRepo.DB()
	patientRepo := repository.NewPatientRepository(db)
	spRepo := repository.NewSchedulePatientRepository(db)
	base.svc.WithPatientRepos(spRepo, patientRepo)
	return &scheduleMultiFixture{
		scheduleFixture: base,
		patientRepo:     patientRepo,
		spRepo:          spRepo,
	}
}

func (fx *scheduleMultiFixture) seedPatient(t *testing.T, name, idNum string) *model.Patient {
	t.Helper()
	p := &model.Patient{Name: name, Phone: "1", IDType: "passport", IDNumber: idNum}
	require.NoError(t, fx.patientRepo.Create(p))
	return p
}

func mkMultiCreateReq(translatorID uint, date string, patients []dto.SchedulePatientPayload) dto.CreateScheduleRequest {
	return dto.CreateScheduleRequest{
		TranslatorID: translatorID,
		Date:         date,
		StartTime:    "09:00",
		EndTime:      "12:00",
		Location:     "Hospital",
		Patients:     patients,
	}
}

func TestScheduleService_Create_WithPatients_Success(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p1 := fx.seedPatient(t, "P1", "PID1")
	p2 := fx.seedPatient(t, "P2", "PID2")

	resp, err := fx.svc.Create(context.Background(), mkMultiCreateReq(fx.translator.ID, "2026-06-10",
		[]dto.SchedulePatientPayload{
			{PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00", PrepaidAmount: 1200},
			{PatientID: p2.ID, StartTime: "10:00", EndTime: "11:00", PrepaidAmount: 0},
		}))
	require.NoError(t, err)
	assert.Len(t, resp.Patients, 2)
	assert.Equal(t, "P1", resp.Patients[0].PatientName)
	assert.Equal(t, "P2", resp.Patients[1].PatientName)
	assert.Equal(t, "pending", resp.Patients[0].Status, "initial status should be pending")
	// Prepaid amount is persisted and returned; actual defaults to 0.
	assert.Equal(t, 1200, resp.Patients[0].PrepaidAmount)
	assert.Equal(t, 0, resp.Patients[0].ActualAmount)
}

func TestScheduleService_Create_WithPatients_EmptyListRejected(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	// Patients explicitly empty AND PatientName not given — should fail.
	req := mkMultiCreateReq(fx.translator.ID, "2026-06-10", []dto.SchedulePatientPayload{})
	_, err := fx.svc.Create(context.Background(), req)
	assert.True(t, errors.Is(err, ErrSchedulePatientsRequired), "expected ErrSchedulePatientsRequired, got %v", err)
}

func TestScheduleService_Create_WithPatients_DuplicatePatientRejected(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p1 := fx.seedPatient(t, "P1", "PID1")
	req := mkMultiCreateReq(fx.translator.ID, "2026-06-10",
		[]dto.SchedulePatientPayload{
			{PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00"},
			{PatientID: p1.ID, StartTime: "10:00", EndTime: "11:00"},
		})
	_, err := fx.svc.Create(context.Background(), req)
	assert.True(t, errors.Is(err, ErrDuplicatePatientInSchedule),
		"expected ErrDuplicatePatientInSchedule, got %v", err)
}

func TestScheduleService_Create_WithPatients_PatientTimeOutOfRange(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p1 := fx.seedPatient(t, "P1", "PID1")
	// schedule 是 09:00-12:00，病人時段 08:00-09:00 落在範圍外
	req := mkMultiCreateReq(fx.translator.ID, "2026-06-10",
		[]dto.SchedulePatientPayload{
			{PatientID: p1.ID, StartTime: "08:00", EndTime: "09:00"},
		})
	_, err := fx.svc.Create(context.Background(), req)
	assert.True(t, errors.Is(err, ErrPatientTimeOutOfRange),
		"expected ErrPatientTimeOutOfRange, got %v", err)
}

func TestScheduleService_Create_WithPatients_PatientNotFound(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	req := mkMultiCreateReq(fx.translator.ID, "2026-06-10",
		[]dto.SchedulePatientPayload{
			{PatientID: 99999, StartTime: "09:00", EndTime: "10:00"},
		})
	_, err := fx.svc.Create(context.Background(), req)
	assert.True(t, errors.Is(err, ErrPatientNotFound),
		"expected ErrPatientNotFound, got %v", err)
}

func TestScheduleService_Create_WithPatients_PatientEndBeforeStartRejected(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p := fx.seedPatient(t, "P", "PID")
	req := mkMultiCreateReq(fx.translator.ID, "2026-06-10",
		[]dto.SchedulePatientPayload{
			{PatientID: p.ID, StartTime: "11:00", EndTime: "10:00"},
		})
	_, err := fx.svc.Create(context.Background(), req)
	assert.True(t, errors.Is(err, ErrPatientEndBeforeStart),
		"expected ErrPatientEndBeforeStart, got %v", err)
}

func TestScheduleService_Update_ReplacesPatients(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p1 := fx.seedPatient(t, "P1", "PID1")
	p2 := fx.seedPatient(t, "P2", "PID2")
	p3 := fx.seedPatient(t, "P3", "PID3")

	// Create with [p1, p2]
	resp, err := fx.svc.Create(context.Background(), mkMultiCreateReq(fx.translator.ID, "2026-06-10",
		[]dto.SchedulePatientPayload{
			{PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00"},
			{PatientID: p2.ID, StartTime: "10:00", EndTime: "11:00"},
		}))
	require.NoError(t, err)

	// Update to [p2, p3]
	newPatients := []dto.SchedulePatientPayload{
		{PatientID: p2.ID, StartTime: "10:00", EndTime: "11:00"},
		{PatientID: p3.ID, StartTime: "11:00", EndTime: "12:00"},
	}
	updated, err := fx.svc.Update(context.Background(), resp.ID, dto.UpdateScheduleRequest{
		Patients: &newPatients,
	})
	require.NoError(t, err)
	require.Len(t, updated.Patients, 2)
	names := []string{updated.Patients[0].PatientName, updated.Patients[1].PatientName}
	assert.ElementsMatch(t, []string{"P2", "P3"}, names)
}

// Regression: deleting a schedule with attached schedule_patients used to
// hit FK constraint "fk_schedules_patients". Cascade must remove SP rows too.
func TestScheduleService_Delete_CascadesSchedulePatients(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p1 := fx.seedPatient(t, "P1", "PID1")

	resp, err := fx.svc.Create(context.Background(), mkMultiCreateReq(fx.translator.ID, "2026-06-10",
		[]dto.SchedulePatientPayload{
			{PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00"},
		}))
	require.NoError(t, err)

	// Should not fail with FK error.
	require.NoError(t, fx.svc.Delete(context.Background(), resp.ID))

	// schedule_patients rows for this schedule should be gone too.
	left, _ := fx.spRepo.FindByScheduleID(resp.ID)
	assert.Empty(t, left, "schedule_patients should be cascade-deleted")
}

func TestScheduleService_DeleteRecurrenceGroup_CascadesSchedulePatients(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p1 := fx.seedPatient(t, "P1", "PID1")

	req := mkMultiCreateReq(fx.translator.ID, "2026-06-01",
		[]dto.SchedulePatientPayload{{PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00"}})
	req.RecurrenceRule = "daily"
	req.RecurrenceUntil = "2026-06-03"
	resp, err := fx.svc.Create(context.Background(), req)
	require.NoError(t, err)

	count, err := fx.svc.DeleteRecurrenceGroup(context.Background(), resp.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 3, count)

	// All groups' schedule_patients should be gone.
	list, _, err := fx.svc.List(context.Background(), fx.translator.ID, "", "", "", 0, 0)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestScheduleService_BackwardCompat_PatientNameStillWorks(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	// 沒給 Patients，但給了 PatientName → 應該照舊舉動
	req := dto.CreateScheduleRequest{
		TranslatorID: fx.translator.ID,
		Date:         "2026-06-10",
		StartTime:    "09:00",
		EndTime:      "12:00",
		Location:     "Hospital",
		PatientName:  "LegacyPatient",
	}
	resp, err := fx.svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "LegacyPatient", resp.PatientName)
	assert.Empty(t, resp.Patients, "legacy single-patient path should not populate Patients[]")
}

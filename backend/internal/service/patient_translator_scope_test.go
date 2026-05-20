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

// patientScopeFixture wires up a PatientService that knows which patients
// belong to a given translator (via SchedulePatientRepository).
type patientScopeFixture struct {
	patientSvc   *PatientService
	scheduleSvc  *ScheduleService
	patientRepo  *repository.PatientRepository
	scheduleRepo *repository.ScheduleRepository
	userRepo     *repository.UserRepository
	spRepo       *repository.SchedulePatientRepository
}

func newPatientScopeFixture(t *testing.T) *patientScopeFixture {
	t.Helper()
	db := newTestDB(t)
	patientRepo := repository.NewPatientRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)
	checkinRepo := repository.NewCheckinRepository(db)
	userRepo := repository.NewUserRepository(db)
	spRepo := repository.NewSchedulePatientRepository(db)

	scheduleSvc := NewScheduleService(scheduleRepo, checkinRepo, userRepo).
		WithPatientRepos(spRepo, patientRepo)
	patientSvc := NewPatientService(patientRepo).WithScopeRepo(spRepo)

	return &patientScopeFixture{
		patientSvc:   patientSvc,
		scheduleSvc:  scheduleSvc,
		patientRepo:  patientRepo,
		scheduleRepo: scheduleRepo,
		userRepo:     userRepo,
		spRepo:       spRepo,
	}
}

func TestPatientService_ListForTranslator_OnlyOwnPatients(t *testing.T) {
	fx := newPatientScopeFixture(t)
	ctx := context.Background()

	// Two translators
	t1 := &model.User{Email: "t1@x.com", PasswordHash: "h", Name: "T1", Role: "translator", Status: "active"}
	t2 := &model.User{Email: "t2@x.com", PasswordHash: "h", Name: "T2", Role: "translator", Status: "active"}
	require.NoError(t, fx.userRepo.Create(t1))
	require.NoError(t, fx.userRepo.Create(t2))

	// Three patients
	p1, _ := fx.patientSvc.Create(ctx, dto.CreatePatientRequest{Name: "P1", Phone: "1", IDType: "passport", IDNumber: "X1"})
	p2, _ := fx.patientSvc.Create(ctx, dto.CreatePatientRequest{Name: "P2", Phone: "2", IDType: "passport", IDNumber: "X2"})
	p3, _ := fx.patientSvc.Create(ctx, dto.CreatePatientRequest{Name: "P3", Phone: "3", IDType: "passport", IDNumber: "X3"})

	// t1 has schedule with p1 + p2; t2 has schedule with p3
	_, err := fx.scheduleSvc.Create(ctx, dto.CreateScheduleRequest{
		TranslatorID: t1.ID, Date: "2026-06-01", StartTime: "09:00", EndTime: "12:00", Location: "L",
		Patients: []dto.SchedulePatientPayload{
			{PatientID: p1.ID, StartTime: "09:00", EndTime: "10:00"},
			{PatientID: p2.ID, StartTime: "10:00", EndTime: "11:00"},
		},
	})
	require.NoError(t, err)
	_, err = fx.scheduleSvc.Create(ctx, dto.CreateScheduleRequest{
		TranslatorID: t2.ID, Date: "2026-06-02", StartTime: "09:00", EndTime: "12:00", Location: "L",
		Patients: []dto.SchedulePatientPayload{
			{PatientID: p3.ID, StartTime: "09:00", EndTime: "10:00"},
		},
	})
	require.NoError(t, err)

	// t1 sees only p1 + p2; t2 sees only p3
	t1List, total, err := fx.patientSvc.ListForTranslator(ctx, t1.ID, dto.PatientListQuery{Page: 1, PageSize: 50})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	names := []string{t1List[0].Name, t1List[1].Name}
	assert.ElementsMatch(t, []string{"P1", "P2"}, names)

	t2List, total, err := fx.patientSvc.ListForTranslator(ctx, t2.ID, dto.PatientListQuery{Page: 1, PageSize: 50})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, "P3", t2List[0].Name)
}

func TestPatientService_ListForTranslator_NoSchedule_ReturnsEmpty(t *testing.T) {
	fx := newPatientScopeFixture(t)
	ctx := context.Background()
	tr := &model.User{Email: "t@x.com", PasswordHash: "h", Name: "T", Role: "translator", Status: "active"}
	require.NoError(t, fx.userRepo.Create(tr))

	list, total, err := fx.patientSvc.ListForTranslator(ctx, tr.ID, dto.PatientListQuery{Page: 1, PageSize: 50})
	require.NoError(t, err)
	assert.Empty(t, list)
	assert.EqualValues(t, 0, total)
}

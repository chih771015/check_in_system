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

// checkinFixture wires up a CheckinService with one translator + one schedule
// so individual tests can focus on the Checkin call shape.
type checkinFixture struct {
	svc          *CheckinService
	checkinRepo  *repository.CheckinRepository
	scheduleRepo *repository.ScheduleRepository
	userRepo     *repository.UserRepository
	translator   *model.User
	schedule     *model.Schedule
}

func newCheckinFixture(t *testing.T) *checkinFixture {
	t.Helper()
	db := newTestDB(t)
	ckRepo := repository.NewCheckinRepository(db)
	schRepo := repository.NewScheduleRepository(db)
	userRepo := repository.NewUserRepository(db)

	tr := &model.User{
		Email: "tr@x.com", PasswordHash: "h", Name: "T", Role: "translator", Status: "active",
	}
	require.NoError(t, userRepo.Create(tr))

	// Schedule for "today" (local midnight) so checkin within window is on-time.
	// Using time.Now().Truncate is UTC-based and breaks across midnight local.
	n := time.Now()
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.Local)
	sch := &model.Schedule{
		TranslatorID: tr.ID,
		Date:         today,
		StartTime:    "09:00",
		EndTime:      "23:59",
		Location:     "Hospital",
		PatientName:  optionalString("Pat"),
	}
	require.NoError(t, schRepo.Create(sch))

	return &checkinFixture{
		svc:          NewCheckinService(ckRepo, schRepo, userRepo, nil),
		checkinRepo:  ckRepo,
		scheduleRepo: schRepo,
		userRepo:     userRepo,
		translator:   tr,
		schedule:     sch,
	}
}

func TestCheckinService_Checkin_ScheduleNotFound(t *testing.T) {
	fx := newCheckinFixture(t)
	_, err := fx.svc.Checkin(context.Background(), fx.translator.ID, 99999, "arrive",
		25.0, 121.5, "", "/u/s.jpg", "/u/e.jpg", false, "")
	assert.True(t, errors.Is(err, ErrScheduleNotFound))
}

func TestCheckinService_Checkin_ScheduleNotOwned(t *testing.T) {
	fx := newCheckinFixture(t)
	other := &model.User{Email: "o@x.com", PasswordHash: "h", Name: "O", Role: "translator", Status: "active"}
	require.NoError(t, fx.userRepo.Create(other))

	_, err := fx.svc.Checkin(context.Background(), other.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "", "/u/s.jpg", "/u/e.jpg", false, "")
	assert.True(t, errors.Is(err, ErrScheduleNotOwned))
}

func TestCheckinService_Checkin_ArriveSuccess(t *testing.T) {
	fx := newCheckinFixture(t)
	resp, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "Taipei", "/u/s.jpg", "/u/e.jpg", false, "")
	require.NoError(t, err)
	assert.Equal(t, "arrive", resp.Type)
	assert.Equal(t, "Taipei", resp.Address)
	assert.Equal(t, fx.translator.Name, resp.TranslatorName)
	assert.False(t, resp.IsMakeup)
}

// Stage 4: environment photo is no longer required — passing empty envURL should succeed.
func TestCheckinService_Checkin_EmptyEnvURLAllowed(t *testing.T) {
	fx := newCheckinFixture(t)
	resp, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "Taipei", "/u/s.jpg", "", false, "")
	require.NoError(t, err)
	assert.Equal(t, "", resp.EnvironmentURL, "empty envURL should be accepted and persisted as empty")
}

func TestCheckinService_Checkin_DuplicateType(t *testing.T) {
	fx := newCheckinFixture(t)
	_, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)

	_, err = fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	assert.True(t, errors.Is(err, ErrDuplicateCheckin))
}

func TestCheckinService_Checkin_LeaveBeforeArriveBlocked(t *testing.T) {
	fx := newCheckinFixture(t)
	_, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "leave",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	assert.True(t, errors.Is(err, ErrArriveBeforeLeave))
}

func TestCheckinService_Checkin_LeaveAfterArriveSuccess(t *testing.T) {
	fx := newCheckinFixture(t)
	_, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)
	resp, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "leave",
		25.0, 121.5, "x", "/u/s2", "/u/e2", false, "")
	require.NoError(t, err)
	assert.Equal(t, "leave", resp.Type)
}

func TestCheckinService_Checkin_AutoMakeupWhenLate(t *testing.T) {
	fx := newCheckinFixture(t)
	// Set schedule end-time to a minute ago so the checkin is past-window.
	pastEnd := time.Now().Add(-1 * time.Minute).Format("15:04")
	fx.schedule.EndTime = pastEnd
	require.NoError(t, fx.scheduleRepo.Update(fx.schedule))

	resp, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)
	assert.True(t, resp.IsMakeup, "checkin after schedule end should auto-mark makeup")
	assert.NotEmpty(t, resp.MakeupReason, "auto makeup should set a reason")
}

func TestCheckinService_AdminUpdateCheckin_NotFound(t *testing.T) {
	fx := newCheckinFixture(t)
	err := fx.svc.AdminUpdateCheckin(context.Background(), 99999, dto.AdminUpdateCheckinRequest{})
	assert.True(t, errors.Is(err, ErrCheckinNotFound))
}

func TestCheckinService_AdminUpdateCheckin_NoFields(t *testing.T) {
	fx := newCheckinFixture(t)
	resp, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)

	err = fx.svc.AdminUpdateCheckin(context.Background(), resp.ID, dto.AdminUpdateCheckinRequest{})
	assert.True(t, errors.Is(err, ErrNoFieldsToUpdate))
}

func TestCheckinService_AdminUpdateCheckin_AddressUpdate(t *testing.T) {
	fx := newCheckinFixture(t)
	resp, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "old", "/u/s", "/u/e", false, "")
	require.NoError(t, err)

	newAddr := "new address"
	require.NoError(t, fx.svc.AdminUpdateCheckin(context.Background(), resp.ID, dto.AdminUpdateCheckinRequest{
		Address: &newAddr,
	}))

	reloaded, _ := fx.checkinRepo.FindByID(resp.ID)
	assert.Equal(t, "new address", reloaded.Address)
}

func TestCheckinService_AdminDeleteCheckin_NotFound(t *testing.T) {
	fx := newCheckinFixture(t)
	err := fx.svc.AdminDeleteCheckin(context.Background(), 99999)
	assert.True(t, errors.Is(err, ErrCheckinNotFound))
}

func TestCheckinService_AdminDeleteCheckin_Success(t *testing.T) {
	fx := newCheckinFixture(t)
	resp, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)

	require.NoError(t, fx.svc.AdminDeleteCheckin(context.Background(), resp.ID))
	_, err = fx.checkinRepo.FindByID(resp.ID)
	assert.Error(t, err)
}

func TestCheckinService_MyHistory_FiltersByTranslator(t *testing.T) {
	fx := newCheckinFixture(t)
	// 自己一筆
	_, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)

	// 另一翻譯員 + 排班 + 打卡
	other := &model.User{Email: "o@x.com", PasswordHash: "h", Name: "O", Role: "translator", Status: "active"}
	require.NoError(t, fx.userRepo.Create(other))
	nn := time.Now()
	otherToday := time.Date(nn.Year(), nn.Month(), nn.Day(), 0, 0, 0, 0, time.Local)
	otherSch := &model.Schedule{TranslatorID: other.ID, Date: otherToday,
		StartTime: "09:00", EndTime: "23:59", Location: "L", PatientName: optionalString("P")}
	require.NoError(t, fx.scheduleRepo.Create(otherSch))
	_, err = fx.svc.Checkin(context.Background(), other.ID, otherSch.ID, "arrive",
		1, 1, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)

	list, err := fx.svc.MyHistory(context.Background(), fx.translator.ID, "", "")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, fx.translator.ID, list[0].TranslatorID)
}

func TestCheckinService_MyStats_CountsByType(t *testing.T) {
	fx := newCheckinFixture(t)
	// arrive + leave
	_, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)
	_, err = fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "leave",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)

	stats, err := fx.svc.MyStats(context.Background(), fx.translator.ID, "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 1, stats.ArriveCount)
	assert.Equal(t, 1, stats.LeaveCount)
	assert.Equal(t, 0, stats.MakeupCount)
}

func TestCheckinService_AdminList_FiltersByType(t *testing.T) {
	fx := newCheckinFixture(t)
	_, err := fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "arrive",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)
	_, err = fx.svc.Checkin(context.Background(), fx.translator.ID, fx.schedule.ID, "leave",
		25.0, 121.5, "x", "/u/s", "/u/e", false, "")
	require.NoError(t, err)

	all, err := fx.svc.AdminList(context.Background(), AdminListParams{})
	require.NoError(t, err)
	assert.Len(t, all, 2)

	arriveOnly, err := fx.svc.AdminList(context.Background(), AdminListParams{CheckinType: "arrive"})
	require.NoError(t, err)
	assert.Len(t, arriveOnly, 1)
	assert.Equal(t, "arrive", arriveOnly[0].Type)
}

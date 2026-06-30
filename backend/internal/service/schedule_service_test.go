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

// scheduleFixture bundles a ready-to-use ScheduleService plus the repos it
// depends on (so test code can pre-seed users / checkins).
type scheduleFixture struct {
	svc          *ScheduleService
	scheduleRepo *repository.ScheduleRepository
	checkinRepo  *repository.CheckinRepository
	userRepo     *repository.UserRepository
	translator   *model.User
}

func newScheduleFixture(t *testing.T) *scheduleFixture {
	t.Helper()
	db := newTestDB(t)
	scheduleRepo := repository.NewScheduleRepository(db)
	checkinRepo := repository.NewCheckinRepository(db)
	userRepo := repository.NewUserRepository(db)

	tr := &model.User{
		Email:        "tr@example.com",
		PasswordHash: "h",
		Name:         "Translator",
		Role:         "translator",
		Status:       "active",
	}
	require.NoError(t, userRepo.Create(tr))

	return &scheduleFixture{
		svc:          NewScheduleService(scheduleRepo, checkinRepo, userRepo),
		scheduleRepo: scheduleRepo,
		checkinRepo:  checkinRepo,
		userRepo:     userRepo,
		translator:   tr,
	}
}

// ─── expandRecurrenceDates (pure function) ───────────────────────────────────

func TestExpandRecurrenceDates_Daily(t *testing.T) {
	start, _ := time.Parse("2006-01-02", "2026-05-01")
	until, _ := time.Parse("2006-01-02", "2026-05-05")
	dates, err := expandRecurrenceDates(start, until, "daily")
	require.NoError(t, err)
	assert.Len(t, dates, 5)
	assert.Equal(t, "2026-05-01", dates[0].Format("2006-01-02"))
	assert.Equal(t, "2026-05-05", dates[4].Format("2006-01-02"))
}

func TestExpandRecurrenceDates_WeeklyMonWedFri(t *testing.T) {
	// 2026-05-01 is Friday, 2026-05-15 is Friday
	start, _ := time.Parse("2006-01-02", "2026-05-01")
	until, _ := time.Parse("2006-01-02", "2026-05-15")
	dates, err := expandRecurrenceDates(start, until, "weekly:1,3,5")
	require.NoError(t, err)
	// 5/1(Fri), 5/4(Mon), 5/6(Wed), 5/8(Fri), 5/11(Mon), 5/13(Wed), 5/15(Fri) = 7
	assert.Len(t, dates, 7)
}

func TestExpandRecurrenceDates_MonthlyClampsToLastDay(t *testing.T) {
	// 31 in Feb 2026 should clamp to Feb 28
	start, _ := time.Parse("2006-01-02", "2026-01-01")
	until, _ := time.Parse("2006-01-02", "2026-04-30")
	dates, err := expandRecurrenceDates(start, until, "monthly:31")
	require.NoError(t, err)
	formatted := make([]string, len(dates))
	for i, d := range dates {
		formatted[i] = d.Format("2006-01-02")
	}
	assert.Contains(t, formatted, "2026-01-31")
	assert.Contains(t, formatted, "2026-02-28") // clamped
	assert.Contains(t, formatted, "2026-03-31")
	assert.Contains(t, formatted, "2026-04-30") // clamped
	assert.Len(t, dates, 4)
}

func TestExpandRecurrenceDates_UnknownRule(t *testing.T) {
	start, _ := time.Parse("2006-01-02", "2026-05-01")
	until, _ := time.Parse("2006-01-02", "2026-05-05")
	_, err := expandRecurrenceDates(start, until, "yearly")
	assert.Error(t, err)
}

func TestExpandRecurrenceDates_InvalidWeekday(t *testing.T) {
	start, _ := time.Parse("2006-01-02", "2026-05-01")
	until, _ := time.Parse("2006-01-02", "2026-05-05")
	_, err := expandRecurrenceDates(start, until, "weekly:9")
	assert.Error(t, err)
}

// ─── Create ──────────────────────────────────────────────────────────────────

func mkCreateReq(translatorID uint, date string) dto.CreateScheduleRequest {
	return dto.CreateScheduleRequest{
		TranslatorID: translatorID,
		Date:         date,
		StartTime:    "09:00",
		EndTime:      "12:00",
		Location:     "Hospital",
		PatientName:  "Patient",
	}
}

func TestScheduleService_Create_Single_Success(t *testing.T) {
	fx := newScheduleFixture(t)
	resp, err := fx.svc.Create(context.Background(), mkCreateReq(fx.translator.ID, "2026-05-10"))
	require.NoError(t, err)
	assert.Equal(t, "2026-05-10", resp.Date)
	assert.Equal(t, "none", resp.CheckinStatus)
	assert.Nil(t, resp.RecurrenceGroupID)
}

func TestScheduleService_Create_TranslatorNotFound(t *testing.T) {
	fx := newScheduleFixture(t)
	_, err := fx.svc.Create(context.Background(), mkCreateReq(99999, "2026-05-10"))
	assert.True(t, errors.Is(err, ErrTranslatorNotFound))
}

func TestScheduleService_Create_UserIsAdminNotTranslator(t *testing.T) {
	fx := newScheduleFixture(t)
	admin := &model.User{
		Email: "a@x.com", PasswordHash: "h", Name: "A", Role: "admin", Status: "active",
	}
	require.NoError(t, fx.userRepo.Create(admin))

	_, err := fx.svc.Create(context.Background(), mkCreateReq(admin.ID, "2026-05-10"))
	assert.True(t, errors.Is(err, ErrNotATranslator))
}

func TestScheduleService_Create_InvalidDate(t *testing.T) {
	fx := newScheduleFixture(t)
	_, err := fx.svc.Create(context.Background(), mkCreateReq(fx.translator.ID, "not-a-date"))
	assert.True(t, errors.Is(err, ErrInvalidDateFormat))
}

func TestScheduleService_Create_Recurring_UntilRequired(t *testing.T) {
	fx := newScheduleFixture(t)
	req := mkCreateReq(fx.translator.ID, "2026-05-01")
	req.RecurrenceRule = "daily"
	_, err := fx.svc.Create(context.Background(), req)
	assert.True(t, errors.Is(err, ErrRecurrenceUntilReq))
}

func TestScheduleService_Create_Recurring_UntilBeforeStart(t *testing.T) {
	fx := newScheduleFixture(t)
	req := mkCreateReq(fx.translator.ID, "2026-05-10")
	req.RecurrenceRule = "daily"
	req.RecurrenceUntil = "2026-05-01"
	_, err := fx.svc.Create(context.Background(), req)
	assert.True(t, errors.Is(err, ErrRecurrenceBeforeStart))
}

func TestScheduleService_Create_Recurring_Daily_GeneratesGroup(t *testing.T) {
	fx := newScheduleFixture(t)
	req := mkCreateReq(fx.translator.ID, "2026-05-01")
	req.RecurrenceRule = "daily"
	req.RecurrenceUntil = "2026-05-03"

	resp, err := fx.svc.Create(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp.RecurrenceGroupID)

	// 應建立 3 筆相同 group 的排班
	list, _, err := fx.svc.List(context.Background(), fx.translator.ID, "", "", "", 0, 0)
	require.NoError(t, err)
	assert.Len(t, list, 3)
	groupID := *resp.RecurrenceGroupID
	for _, s := range list {
		require.NotNil(t, s.RecurrenceGroupID)
		assert.Equal(t, groupID, *s.RecurrenceGroupID)
	}
}

// ─── Update ──────────────────────────────────────────────────────────────────

func TestScheduleService_Update_NotFound(t *testing.T) {
	fx := newScheduleFixture(t)
	_, _, err := fx.svc.Update(context.Background(), 99999, dto.UpdateScheduleRequest{})
	assert.True(t, errors.Is(err, ErrScheduleNotFound))
}

func TestScheduleService_Update_PartialField(t *testing.T) {
	fx := newScheduleFixture(t)
	created, err := fx.svc.Create(context.Background(), mkCreateReq(fx.translator.ID, "2026-05-10"))
	require.NoError(t, err)

	newLocation := "New Place"
	resp, _, err := fx.svc.Update(context.Background(), created.ID, dto.UpdateScheduleRequest{
		Location: &newLocation,
	})
	require.NoError(t, err)
	assert.Equal(t, "New Place", resp.Location)
	assert.Equal(t, "09:00", resp.StartTime) // 沒動的欄位保留
}

// ─── Delete / DeleteRecurrenceGroup ──────────────────────────────────────────

func TestScheduleService_Delete_NotFound(t *testing.T) {
	fx := newScheduleFixture(t)
	_, err := fx.svc.Delete(context.Background(), 99999)
	assert.True(t, errors.Is(err, ErrScheduleNotFound))
}

func TestScheduleService_Delete_CascadesCheckins(t *testing.T) {
	fx := newScheduleFixture(t)
	created, err := fx.svc.Create(context.Background(), mkCreateReq(fx.translator.ID, "2026-05-10"))
	require.NoError(t, err)

	// 給該排班 seed 一筆打卡
	ck := &model.Checkin{
		ScheduleID:   created.ID,
		TranslatorID: fx.translator.ID,
		Type:         "arrive",
		CheckinTime:  time.Now(),
		Latitude:     25.0,
		Longitude:    121.5,
		Address:      "x",
		SelfieURL:    "/uploads/s.jpg",
	}
	require.NoError(t, fx.checkinRepo.Create(ck))

	_, err = fx.svc.Delete(context.Background(), created.ID)
	require.NoError(t, err)

	// 排班與打卡都應該不見
	_, err = fx.scheduleRepo.FindByID(created.ID)
	assert.Error(t, err)
	cks, _ := fx.checkinRepo.FindByScheduleID(created.ID)
	assert.Empty(t, cks)
}

func TestScheduleService_DeleteRecurrenceGroup_SingleFallback(t *testing.T) {
	fx := newScheduleFixture(t)
	created, err := fx.svc.Create(context.Background(), mkCreateReq(fx.translator.ID, "2026-05-10"))
	require.NoError(t, err)

	count, _, err := fx.svc.DeleteRecurrenceGroup(context.Background(), created.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count, "single schedule should fall back to delete-one")
}

func TestScheduleService_DeleteRecurrenceGroup_BulkDelete(t *testing.T) {
	fx := newScheduleFixture(t)
	req := mkCreateReq(fx.translator.ID, "2026-05-01")
	req.RecurrenceRule = "daily"
	req.RecurrenceUntil = "2026-05-04"
	first, err := fx.svc.Create(context.Background(), req)
	require.NoError(t, err)

	count, detail, err := fx.svc.DeleteRecurrenceGroup(context.Background(), first.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 4, count)
	// Group delete audit detail should record how many rows were removed.
	assert.Contains(t, detail, "deletedCount")

	list, _, _ := fx.svc.List(context.Background(), fx.translator.ID, "", "", "", 0, 0)
	assert.Empty(t, list, "all grouped schedules should be gone")
}

// ─── getCheckinStatus via List ───────────────────────────────────────────────

func TestScheduleService_List_CheckinStatusPriority(t *testing.T) {
	fx := newScheduleFixture(t)
	s1, _ := fx.svc.Create(context.Background(), mkCreateReq(fx.translator.ID, "2026-05-10"))
	s2, _ := fx.svc.Create(context.Background(), mkCreateReq(fx.translator.ID, "2026-05-11"))
	s3, _ := fx.svc.Create(context.Background(), mkCreateReq(fx.translator.ID, "2026-05-12"))

	// s1: 沒打卡 → none
	// s2: 只有 arrive → arrived
	require.NoError(t, fx.checkinRepo.Create(&model.Checkin{
		ScheduleID: s2.ID, TranslatorID: fx.translator.ID, Type: "arrive",
		CheckinTime: time.Now(), Latitude: 1, Longitude: 1, Address: "x", SelfieURL: "/x",
	}))
	// s3: arrive + leave → completed（即使是 makeup 也是 completed，因為兩個都有）
	require.NoError(t, fx.checkinRepo.Create(&model.Checkin{
		ScheduleID: s3.ID, TranslatorID: fx.translator.ID, Type: "arrive", IsMakeup: true,
		CheckinTime: time.Now(), Latitude: 1, Longitude: 1, Address: "x", SelfieURL: "/x",
	}))
	require.NoError(t, fx.checkinRepo.Create(&model.Checkin{
		ScheduleID: s3.ID, TranslatorID: fx.translator.ID, Type: "leave", IsMakeup: true,
		CheckinTime: time.Now(), Latitude: 1, Longitude: 1, Address: "x", SelfieURL: "/x",
	}))

	list, _, err := fx.svc.List(context.Background(), fx.translator.ID, "", "", "", 0, 0)
	require.NoError(t, err)
	statusByID := map[uint]string{}
	for _, s := range list {
		statusByID[s.ID] = s.CheckinStatus
	}
	assert.Equal(t, "none", statusByID[s1.ID])
	assert.Equal(t, "arrived", statusByID[s2.ID])
	assert.Equal(t, "completed", statusByID[s3.ID])
}

func TestScheduleService_List_DefaultRecentWhenUnfiltered(t *testing.T) {
	fx := newScheduleFixture(t)
	ctx := context.Background()
	s1, _ := fx.svc.Create(ctx, mkCreateReq(fx.translator.ID, "2026-05-10"))
	s2, _ := fx.svc.Create(ctx, mkCreateReq(fx.translator.ID, "2026-05-11"))
	s3, _ := fx.svc.Create(ctx, mkCreateReq(fx.translator.ID, "2026-05-12"))

	// No filter → default mode: most recently created first (created_at DESC),
	// which is the reverse of date ASC. Created order s1,s2,s3 → expect s3,s2,s1.
	list, _, err := fx.svc.List(ctx, 0, "", "", "", 0, 0)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, s3.ID, list[0].ID)
	assert.Equal(t, s2.ID, list[1].ID)
	assert.Equal(t, s1.ID, list[2].ID)
}

func TestScheduleService_List_Paginates(t *testing.T) {
	fx := newScheduleFixture(t)
	ctx := context.Background()
	fx.svc.Create(ctx, mkCreateReq(fx.translator.ID, "2026-05-10"))
	fx.svc.Create(ctx, mkCreateReq(fx.translator.ID, "2026-05-11"))
	fx.svc.Create(ctx, mkCreateReq(fx.translator.ID, "2026-05-12"))

	// Default (unfiltered) mode, page 1 of size 2.
	page1, total, err := fx.svc.List(ctx, 0, "", "", "", 1, 2)
	require.NoError(t, err)
	assert.Len(t, page1, 2, "page size caps rows")
	assert.Equal(t, int64(3), total, "total is the full count")

	page2, _, err := fx.svc.List(ctx, 0, "", "", "", 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 1, "page 2 has the leftover row")

	// Filtered mode also paginates and reports the full matching total.
	fp1, ftotal, err := fx.svc.List(ctx, 0, "2026-05-01", "2026-05-31", "", 1, 2)
	require.NoError(t, err)
	assert.Len(t, fp1, 2)
	assert.Equal(t, int64(3), ftotal)
}

func TestScheduleService_List_FilteredUsesDateOrder(t *testing.T) {
	fx := newScheduleFixture(t)
	ctx := context.Background()
	// Created later-date first so created order differs from date order.
	later, _ := fx.svc.Create(ctx, mkCreateReq(fx.translator.ID, "2026-05-12"))
	earlier, _ := fx.svc.Create(ctx, mkCreateReq(fx.translator.ID, "2026-05-10"))

	// A date-range filter → filtered mode keeps the existing date ASC ordering.
	list, _, err := fx.svc.List(ctx, 0, "2026-05-01", "2026-05-31", "", 0, 0)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, earlier.ID, list[0].ID, "filtered mode orders by date ASC")
	assert.Equal(t, later.ID, list[1].ID)
}

// ─── BatchImportSchedules ────────────────────────────────────────────────────

func TestScheduleService_BatchImport_MixedResults(t *testing.T) {
	fx := newScheduleFixture(t)

	rows := []ScheduleImportRow{
		// 1: pre-existing error from upstream → skip
		{RowNumber: 2, Error: "translatorId is empty"},
		// 2: invalid date
		{RowNumber: 3, TranslatorID: fx.translator.ID, Date: "bad",
			StartTime: "09:00", EndTime: "10:00", Location: "L", PatientName: "P"},
		// 3: translator not found
		{RowNumber: 4, TranslatorID: 99999, Date: "2026-05-10",
			StartTime: "09:00", EndTime: "10:00", Location: "L", PatientName: "P"},
		// 4: success
		{RowNumber: 5, TranslatorID: fx.translator.ID, Date: "2026-05-11",
			StartTime: "09:00", EndTime: "10:00", Location: "L", PatientName: "P"},
		// 5: success
		{RowNumber: 6, TranslatorID: fx.translator.ID, Date: "2026-05-12",
			StartTime: "09:00", EndTime: "10:00", Location: "L", PatientName: "P"},
	}

	success, failed := fx.svc.BatchImportSchedules(context.Background(), rows)
	assert.Equal(t, 2, success)
	assert.Len(t, failed, 3)
	// 失敗的應該保留行號讓使用者定位
	failedRowNums := []int{failed[0].RowNumber, failed[1].RowNumber, failed[2].RowNumber}
	assert.ElementsMatch(t, []int{2, 3, 4}, failedRowNums)
}

func TestScheduleService_BatchImport_NonTranslatorRejected(t *testing.T) {
	fx := newScheduleFixture(t)
	admin := &model.User{
		Email: "a@x.com", PasswordHash: "h", Name: "A", Role: "admin", Status: "active",
	}
	require.NoError(t, fx.userRepo.Create(admin))

	rows := []ScheduleImportRow{
		{RowNumber: 2, TranslatorID: admin.ID, Date: "2026-05-10",
			StartTime: "09:00", EndTime: "10:00", Location: "L", PatientName: "P"},
	}
	success, failed := fx.svc.BatchImportSchedules(context.Background(), rows)
	assert.Equal(t, 0, success)
	require.Len(t, failed, 1)
	assert.Equal(t, "translator not found", failed[0].Error)
}

package service

import (
	"context"
	"testing"

	"translator-checkin/internal/config"
	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newExportFixture(t *testing.T) (*ExportService, *checkinFixture, *repository.ExportScheduleRepository) {
	t.Helper()
	db := newTestDB(t)
	scheduleRepo := repository.NewScheduleRepository(db)
	checkinRepo := repository.NewCheckinRepository(db)
	userRepo := repository.NewUserRepository(db)
	exportRepo := repository.NewExportScheduleRepository(db)
	checkinSvc := NewCheckinService(checkinRepo, scheduleRepo, userRepo, nil)
	mailSvc := NewMailService()

	tr := &model.User{Email: "t@x.com", PasswordHash: "h", Name: "T", Role: "translator", Status: "active"}
	require.NoError(t, userRepo.Create(tr))

	return NewExportService(checkinSvc, exportRepo, mailSvc),
		&checkinFixture{
			svc:          checkinSvc,
			checkinRepo:  checkinRepo,
			scheduleRepo: scheduleRepo,
			userRepo:     userRepo,
			translator:   tr,
		},
		exportRepo
}

func TestExportService_CheckinRow_TypeAndMakeupLabels(t *testing.T) {
	row := checkinRow(dto.CheckinResponse{
		ID: 1, TranslatorID: 2, TranslatorName: "Alice", Type: "arrive",
		Address: "Hospital", Latitude: 25.0, Longitude: 121.5,
		SelfieURL: "/s", EnvironmentURL: "/e", IsMakeup: false, MakeupReason: "",
	})
	assert.Equal(t, "到達", row[3])
	assert.Equal(t, "否", row[10])

	row = checkinRow(dto.CheckinResponse{Type: "leave", IsMakeup: true})
	assert.Equal(t, "離開", row[3])
	assert.Equal(t, "是", row[10])
}

func TestExportService_BuildCheckinExcel_EmptyAndOne(t *testing.T) {
	svc, _, _ := newExportFixture(t)

	// Empty data
	f, err := svc.BuildCheckinExcel(context.Background(), AdminListParams{})
	require.NoError(t, err)
	require.NotNil(t, f)
	sheets := f.GetSheetList()
	assert.Contains(t, sheets, "打卡紀錄")

	// header on row 1 col 1
	header, err := f.GetCellValue("打卡紀錄", "A1")
	require.NoError(t, err)
	assert.Equal(t, "打卡ID", header)
}

func TestExportService_CreateCheckinGoogleSheet_NoCreds(t *testing.T) {
	initTestConfig()
	prev := *config.AppConfig
	defer func() { *config.AppConfig = prev }()
	config.AppConfig.GoogleCredentialsFile = ""

	svc, _, _ := newExportFixture(t)
	_, err := svc.CreateCheckinGoogleSheet(context.Background(), AdminListParams{}, "title")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Google credentials not configured")
}

func TestExportService_RunExportForAdmin_NoSchedule(t *testing.T) {
	svc, _, _ := newExportFixture(t)
	_, err := svc.RunExportForAdmin(context.Background(), 99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "export schedule not found")
}

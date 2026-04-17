package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"translator-checkin/internal/config"
	"translator-checkin/internal/dto"
	"translator-checkin/internal/repository"

	"github.com/xuri/excelize/v2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// ExportService produces checkin reports (Excel or Google Sheet) and is
// shared by manual export endpoints, the admin Excel/GoogleSheet handlers,
// and the periodic export cron.
type ExportService struct {
	checkinService     *CheckinService
	exportScheduleRepo *repository.ExportScheduleRepository
	mailService        *MailService
}

// NewExportService creates a new ExportService.
func NewExportService(
	checkinService *CheckinService,
	exportScheduleRepo *repository.ExportScheduleRepository,
	mailService *MailService,
) *ExportService {
	return &ExportService{
		checkinService:     checkinService,
		exportScheduleRepo: exportScheduleRepo,
		mailService:        mailService,
	}
}

var checkinExportHeaders = []interface{}{
	"打卡ID", "翻譯員ID", "翻譯員姓名", "打卡類型", "打卡時間", "地址",
	"GPS緯度", "GPS經度", "自拍照URL", "環境照URL", "是否補打卡", "補打卡原因",
}

func checkinRow(ck dto.CheckinResponse) []interface{} {
	typeLabel := "到達"
	if ck.Type == "leave" {
		typeLabel = "離開"
	}
	makeupLabel := "否"
	if ck.IsMakeup {
		makeupLabel = "是"
	}
	return []interface{}{
		ck.ID, ck.TranslatorID, ck.TranslatorName, typeLabel,
		ck.CheckinTime.Format("2006-01-02 15:04:05"),
		ck.Address, ck.Latitude, ck.Longitude,
		ck.SelfieURL, ck.EnvironmentURL, makeupLabel, ck.MakeupReason,
	}
}

// BuildCheckinExcel queries checkins matching params and returns an in-memory
// Excel file.
func (s *ExportService) BuildCheckinExcel(ctx context.Context, params AdminListParams) (*excelize.File, error) {
	checkins, err := s.checkinService.AdminList(ctx, params)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheet := "打卡紀錄"
	f.NewSheet(sheet)
	f.DeleteSheet("Sheet1")

	for i, h := range checkinExportHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for rowIdx, ck := range checkins {
		row := rowIdx + 2
		for colIdx, val := range checkinRow(ck) {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			f.SetCellValue(sheet, cell, val)
		}
	}
	return f, nil
}

// CreateCheckinGoogleSheet creates a new Google Sheet populated with checkins
// matching params and returns the sheet URL.
func (s *ExportService) CreateCheckinGoogleSheet(ctx context.Context, params AdminListParams, title string) (string, error) {
	credFile := config.AppConfig.GoogleCredentialsFile
	if credFile == "" {
		return "", errors.New("Google credentials not configured (set GOOGLE_CREDENTIALS_FILE env var)")
	}

	if title == "" {
		title = fmt.Sprintf("打卡紀錄_%s", time.Now().Format("20060102_150405"))
	}

	checkins, err := s.checkinService.AdminList(ctx, params)
	if err != nil {
		return "", err
	}

	credBytes, err := os.ReadFile(credFile)
	if err != nil {
		return "", fmt.Errorf("failed to read credentials file: %w", err)
	}
	conf, err := google.JWTConfigFromJSON(credBytes, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return "", fmt.Errorf("invalid credentials: %w", err)
	}
	client := conf.Client(ctx)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", fmt.Errorf("failed to create Sheets client: %w", err)
	}

	spreadsheet, err := srv.Spreadsheets.Create(&sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{Title: title},
	}).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create spreadsheet: %w", err)
	}

	rows := [][]interface{}{checkinExportHeaders}
	for _, ck := range checkins {
		rows = append(rows, checkinRow(ck))
	}

	vr := &sheets.ValueRange{Values: rows}
	_, err = srv.Spreadsheets.Values.Update(spreadsheet.SpreadsheetId, "Sheet1!A1", vr).
		ValueInputOption("RAW").Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to write data: %w", err)
	}

	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", spreadsheet.SpreadsheetId), nil
}

// RunResult summarizes a single export execution.
type RunResult struct {
	Format    string    `json:"format"`
	EmailTo   string    `json:"emailTo"`
	SheetURL  string    `json:"sheetUrl,omitempty"`
	Filename  string    `json:"filename,omitempty"`
	RangeFrom string    `json:"rangeFrom"`
	RangeTo   string    `json:"rangeTo"`
	RanAt     time.Time `json:"ranAt"`
}

// RunExportForAdmin executes the configured periodic export for one admin:
// builds the report (Excel or Google Sheet) for the previous calendar month,
// emails it to EmailTo, and updates last_run_at on success.
func (s *ExportService) RunExportForAdmin(ctx context.Context, adminID uint) (*RunResult, error) {
	es, err := s.exportScheduleRepo.WithCtx(ctx).FindByAdmin(adminID)
	if err != nil {
		return nil, errors.New("export schedule not found for this admin")
	}
	if es.EmailTo == "" {
		return nil, errors.New("emailTo is empty; configure a recipient before running")
	}

	now := time.Now()
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastOfPrevMonth := firstOfThisMonth.AddDate(0, 0, -1)
	firstOfPrevMonth := time.Date(lastOfPrevMonth.Year(), lastOfPrevMonth.Month(), 1, 0, 0, 0, 0, now.Location())
	from := firstOfPrevMonth.Format("2006-01-02")
	to := lastOfPrevMonth.Format("2006-01-02")

	params := AdminListParams{DateFrom: from, DateTo: to}

	result := &RunResult{
		Format:    es.Format,
		EmailTo:   es.EmailTo,
		RangeFrom: from,
		RangeTo:   to,
		RanAt:     now,
	}

	subject := fmt.Sprintf("打卡紀錄報表 %s ~ %s", from, to)

	switch es.Format {
	case "excel":
		f, err := s.BuildCheckinExcel(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("build excel: %w", err)
		}
		var buf bytes.Buffer
		if err := f.Write(&buf); err != nil {
			return nil, fmt.Errorf("serialize excel: %w", err)
		}
		filename := fmt.Sprintf("checkins_%s_%s.xlsx", from, to)
		body := fmt.Sprintf("附件為 %s 至 %s 的打卡紀錄。", from, to)
		err = s.mailService.Send(es.EmailTo, subject, body, &Attachment{
			Filename:    filename,
			ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			Data:        buf.Bytes(),
		})
		if err != nil {
			return nil, fmt.Errorf("send email: %w", err)
		}
		result.Filename = filename

	case "google_sheet":
		title := fmt.Sprintf("打卡紀錄_%s_%s", from, to)
		url, err := s.CreateCheckinGoogleSheet(ctx, params, title)
		if err != nil {
			return nil, fmt.Errorf("create google sheet: %w", err)
		}
		body := fmt.Sprintf("已建立 %s 至 %s 的 Google Sheet：\n%s", from, to, url)
		if err := s.mailService.Send(es.EmailTo, subject, body, nil); err != nil {
			return nil, fmt.Errorf("send email: %w", err)
		}
		result.SheetURL = url

	default:
		return nil, fmt.Errorf("unsupported export format: %s", es.Format)
	}

	if err := s.exportScheduleRepo.WithCtx(ctx).UpdateLastRun(es.ID, now); err != nil {
		return nil, fmt.Errorf("update last_run_at: %w", err)
	}
	return result, nil
}

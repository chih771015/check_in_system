package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ErrPatientDuplicate is returned when a Create/Update would collide with an
// existing patient on (id_type, id_number). Handlers map this to 409 / a
// dedicated error code.
var ErrPatientDuplicate = errors.New("patient with same id_type and id_number already exists")

// ErrPatientNotFound is returned when a patient lookup misses.
var ErrPatientNotFound = errors.New("patient not found")

// PatientService implements CRUD for patients plus the visit-history
// aggregation hook (stub in stage 2, real in stage 4).
type PatientService struct {
	patientRepo *repository.PatientRepository
	// Stage 3: when set, ListForTranslator restricts results to patients
	// the caller actually has in their schedules.
	spRepo *repository.SchedulePatientRepository
	// Stage 4: history aggregation deps.
	scheduleRepo *repository.ScheduleRepository
	photoRepo    *repository.DiagnosisPhotoRepository
}

// NewPatientService creates a new PatientService.
func NewPatientService(patientRepo *repository.PatientRepository) *PatientService {
	return &PatientService{patientRepo: patientRepo}
}

// WithScopeRepo wires up SchedulePatientRepository so ListForTranslator can
// restrict results to the caller's own schedules. Returns the service for
// chaining.
func (s *PatientService) WithScopeRepo(spRepo *repository.SchedulePatientRepository) *PatientService {
	s.spRepo = spRepo
	return s
}

// WithHistoryRepos wires up Schedule + SchedulePatient + DiagnosisPhoto repos
// so GetHistory can return real visit data.
func (s *PatientService) WithHistoryRepos(
	scheduleRepo *repository.ScheduleRepository,
	spRepo *repository.SchedulePatientRepository,
	photoRepo *repository.DiagnosisPhotoRepository,
) *PatientService {
	s.scheduleRepo = scheduleRepo
	s.spRepo = spRepo
	s.photoRepo = photoRepo
	return s
}

// normalizeIDNumber uppercases and trims the ID number so that lookups are
// case- and whitespace-insensitive without changing user-supplied display.
func normalizeIDNumber(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// Create inserts a new patient after checking for duplicates on
// (id_type, id_number). Returns ErrPatientDuplicate if a collision is found.
func (s *PatientService) Create(ctx context.Context, req dto.CreatePatientRequest) (*model.Patient, error) {
	repo := s.patientRepo.WithCtx(ctx)
	idNumber := normalizeIDNumber(req.IDNumber)

	if existing, err := repo.FindByIDTypeAndNumber(req.IDType, idNumber); err == nil && existing != nil {
		return nil, ErrPatientDuplicate
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	patient := &model.Patient{
		Name:     strings.TrimSpace(req.Name),
		Phone:    strings.TrimSpace(req.Phone),
		IDType:   req.IDType,
		IDNumber: idNumber,
	}
	if err := repo.Create(patient); err != nil {
		return nil, err
	}
	return patient, nil
}

// PatientImportRow is one raw row parsed from the import xlsx (handler builds
// these from sheet cells; the service validates + normalises).
type PatientImportRow struct {
	Name     string
	Phone    string
	IDType   string
	IDNumber string
}

var validIDTypes = map[string]bool{"passport": true, "hn": true, "unid": true}

// ImportPatients bulk-creates patients from parsed rows. Duplicates (same
// id_type + id_number) and invalid rows are skipped and reported; valid rows
// are created. Never aborts the whole batch on a single bad row.
func (s *PatientService) ImportPatients(ctx context.Context, rows []PatientImportRow) *dto.PatientImportResult {
	res := &dto.PatientImportResult{Errors: []dto.PatientImportError{}}
	skip := func(row int, reason string) {
		res.Errors = append(res.Errors, dto.PatientImportError{Row: row, Reason: reason})
		res.Skipped++
	}
	for i, r := range rows {
		sheetRow := i + 2 // header occupies row 1
		name := strings.TrimSpace(r.Name)
		phone := strings.TrimSpace(r.Phone)
		idType := strings.ToLower(strings.TrimSpace(r.IDType))
		idNumber := strings.TrimSpace(r.IDNumber)

		// Fully-blank row (e.g. a spacer line in the sheet) → skip silently,
		// don't count or report it. Row numbers stay correct because we keep
		// iterating the full slice.
		if name == "" && phone == "" && idType == "" && idNumber == "" {
			continue
		}
		if name == "" || phone == "" || idNumber == "" {
			skip(sheetRow, "缺少必填欄位（姓名/電話/證件號碼）")
			continue
		}
		if !validIDTypes[idType] {
			skip(sheetRow, "證件類型非法（須為 passport / hn / unid）")
			continue
		}

		_, err := s.Create(ctx, dto.CreatePatientRequest{Name: name, Phone: phone, IDType: idType, IDNumber: idNumber})
		switch {
		case errors.Is(err, ErrPatientDuplicate):
			skip(sheetRow, "重複（證件類型 + 號碼已存在）")
		case err != nil:
			skip(sheetRow, err.Error())
		default:
			res.Created++
		}
	}
	return res
}

var patientExcelHeaders = []interface{}{"姓名", "電話", "證件類型(passport/hn/unid)", "證件號碼"}

// BuildExcel returns an in-memory xlsx of all patients. The first 4 columns
// match the import format (姓名/電話/證件類型/證件號碼) so the file round-trips
// through import; an export-only 5th column "實付金額總額" carries each patient's
// all-time actual-paid total. The import header/parsing stays at 4 columns, so
// adding this column does not break re-importing the exported file.
func (s *PatientService) BuildExcel(ctx context.Context) (*excelize.File, error) {
	patients, _, err := s.patientRepo.WithCtx(ctx).List("", 1, 1000000)
	if err != nil {
		return nil, err
	}

	// Batch-fetch actual-paid totals (avoids N+1). When the scope repo is not
	// wired (legacy callers) totals default to 0.
	totals := map[uint]int64{}
	if s.spRepo != nil && len(patients) > 0 {
		ids := make([]uint, len(patients))
		for i, p := range patients {
			ids[i] = p.ID
		}
		totals, err = s.spRepo.WithCtx(ctx).SumActualByPatients(ids)
		if err != nil {
			return nil, err
		}
	}

	f := newPatientSheet()
	sheet := f.GetSheetName(0)
	// Export-only header in column E (not part of the import header constant).
	f.SetCellValue(sheet, "E1", "實付金額總額")
	for rowIdx, p := range patients {
		for colIdx, val := range []interface{}{p.Name, p.Phone, p.IDType, p.IDNumber} {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheet, cell, val)
		}
		totalCell, _ := excelize.CoordinatesToCellName(5, rowIdx+2)
		f.SetCellValue(sheet, totalCell, totals[p.ID])
	}
	return f, nil
}

// BuildPatientTemplate returns an xlsx with the header and one example row per
// id type (passport / hn / unid) to guide bulk import.
func BuildPatientTemplate() *excelize.File {
	f := newPatientSheet()
	sheet := f.GetSheetName(0)
	examples := [][]interface{}{
		{"王小明", "0912345678", "passport", "A1234567"},
		{"林春嬌", "0922333444", "hn", "HN000123"},
		{"無名氏", "0900000000", "unid", "UN-XYZ-01"},
	}
	for r, row := range examples {
		for colIdx, val := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, r+2)
			f.SetCellValue(sheet, cell, val)
		}
	}
	return f
}

// newPatientSheet builds a single-sheet xlsx with the patient header row.
func newPatientSheet() *excelize.File {
	f := excelize.NewFile()
	sheet := "病人"
	f.NewSheet(sheet)
	f.DeleteSheet("Sheet1")
	for i, h := range patientExcelHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	return f
}

// Update edits an existing patient. The duplicate check ignores the current
// record so a no-op update still works.
// Update modifies a patient and returns the updated record plus an audit detail
// JSON describing the before/after state.
func (s *PatientService) Update(ctx context.Context, id uint, req dto.UpdatePatientRequest) (*model.Patient, string, error) {
	repo := s.patientRepo.WithCtx(ctx)
	patient, err := repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrPatientNotFound
		}
		return nil, "", err
	}

	before := snapshotPatient(patient)

	idNumber := normalizeIDNumber(req.IDNumber)
	if existing, err := repo.FindByIDTypeAndNumber(req.IDType, idNumber); err == nil && existing != nil && existing.ID != id {
		return nil, "", ErrPatientDuplicate
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}

	patient.Name = strings.TrimSpace(req.Name)
	patient.Phone = strings.TrimSpace(req.Phone)
	patient.IDType = req.IDType
	patient.IDNumber = idNumber

	if err := repo.Update(patient); err != nil {
		return nil, "", err
	}
	return patient, auditDetailJSON(before, snapshotPatient(patient)), nil
}

// Delete removes a patient by ID and returns an audit detail JSON containing a
// snapshot of the deleted record.
func (s *PatientService) Delete(ctx context.Context, id uint) (string, error) {
	repo := s.patientRepo.WithCtx(ctx)
	patient, err := repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrPatientNotFound
		}
		return "", err
	}
	if err := repo.Delete(id); err != nil {
		return "", err
	}
	return auditDetailJSON(snapshotPatient(patient), nil), nil
}

// FindByID exposes the underlying repository lookup so handlers don't have to
// touch the repo directly.
func (s *PatientService) FindByID(ctx context.Context, id uint) (*model.Patient, error) {
	repo := s.patientRepo.WithCtx(ctx)
	patient, err := repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPatientNotFound
		}
		return nil, err
	}
	return patient, nil
}

// List returns a page of patients with total count.
func (s *PatientService) List(ctx context.Context, q dto.PatientListQuery) ([]model.Patient, int64, error) {
	return s.patientRepo.WithCtx(ctx).List(strings.TrimSpace(q.Search), q.Page, q.PageSize)
}

// ListForTranslator returns the patient list a translator may see — only
// patients that appear in their own schedules (via schedule_patients).
//
// When WithScopeRepo has not been called the service falls back to listing
// all patients (legacy stage-2 behaviour for tests that don't wire scope).
func (s *PatientService) ListForTranslator(ctx context.Context, translatorID uint, q dto.PatientListQuery) ([]model.Patient, int64, error) {
	if s.spRepo == nil {
		return s.patientRepo.WithCtx(ctx).List(strings.TrimSpace(q.Search), q.Page, q.PageSize)
	}
	return s.patientRepo.WithCtx(ctx).ListForTranslator(translatorID, strings.TrimSpace(q.Search), q.Page, q.PageSize)
}

// ActualTotals returns the all-time actual_amount sum per patient id (missing
// = 0). Used by the admin patient list to show each patient's實付總額 in one
// batched query. Returns an empty map when the scope repo is not wired.
func (s *PatientService) ActualTotals(ctx context.Context, patientIDs []uint) (map[uint]int64, error) {
	if s.spRepo == nil {
		return map[uint]int64{}, nil
	}
	return s.spRepo.WithCtx(ctx).SumActualByPatients(patientIDs)
}

// PatientYearActualTotal returns the sum of actual_amount for a patient over
// the given calendar year (by schedule date). Shown at scheduling time so staff
// can see how much the patient has already been paid that year. Returns 0 when
// the scope repo is not wired.
func (s *PatientService) PatientYearActualTotal(ctx context.Context, patientID uint, year int) (int64, error) {
	if s.spRepo == nil {
		return 0, nil
	}
	from := fmt.Sprintf("%04d-01-01", year)
	to := fmt.Sprintf("%04d-01-01", year+1)
	return s.spRepo.WithCtx(ctx).SumActualByPatientDateRange(patientID, from, to)
}

// GetHistory returns a patient's visit history aggregated from
// schedule_patients + schedules + diagnosis_photos, ordered by date DESC.
//
// When from/to (YYYY-MM-DD) are supplied the history is filtered to that
// inclusive date range; ActualTotal in the response is the sum of actual_amount
// over the returned entries (range total when filtered, all-time otherwise).
//
// If history repos have not been wired (legacy stage-2 caller) it falls back to
// an empty slice.
func (s *PatientService) GetHistory(ctx context.Context, patientID uint, from, to string) (*dto.PatientHistoryResponse, error) {
	patient, err := s.FindByID(ctx, patientID)
	if err != nil {
		return nil, err
	}

	// Reject malformed date filters up front so a bad bound fails closed (400)
	// rather than being silently dropped or compared lexicographically. Empty
	// means "no bound".
	if !validDateOrEmpty(from) || !validDateOrEmpty(to) {
		return nil, ErrInvalidDateFormat
	}

	// The date range (when supplied) is pushed into SQL by buildHistoryEntries,
	// so we only fetch the rows we keep; sum actual_amount over them here.
	entries := []dto.PatientHistoryEntry{}
	if s.scheduleRepo != nil && s.spRepo != nil && s.photoRepo != nil {
		entries, err = s.buildHistoryEntries(ctx, patientID, from, to)
		if err != nil {
			return nil, err
		}
	}

	var total int64
	for _, e := range entries {
		total += int64(e.ActualAmount)
	}

	return &dto.PatientHistoryResponse{
		Patient: dto.PatientResponse{
			ID:        patient.ID,
			Name:      patient.Name,
			Phone:     patient.Phone,
			IDType:    patient.IDType,
			IDNumber:  patient.IDNumber,
			CreatedAt: patient.CreatedAt,
			UpdatedAt: patient.UpdatedAt,
		},
		History:     entries,
		ActualTotal: total,
	}, nil
}

// validDateOrEmpty reports whether s is empty (no bound) or a valid YYYY-MM-DD
// date. Used to reject malformed history-range filters before they reach SQL.
func validDateOrEmpty(s string) bool {
	if s == "" {
		return true
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// nextDay returns the calendar day after a YYYY-MM-DD date, as YYYY-MM-DD. It
// turns an inclusive upper bound into a half-open exclusive one for SQL date
// comparison — safe across sqlite's RFC3339-stored dates and postgres date
// columns, where a plain `<= to` would drop same-day rows stored as
// "YYYY-MM-DDT00:00:00Z". ok is false when date is not a valid YYYY-MM-DD.
func nextDay(date string) (string, bool) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", false
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02"), true
}

// buildHistoryEntries does the real DB walk for GetHistory. When from/to
// (YYYY-MM-DD) are supplied the date range is applied in SQL (inclusive of both
// ends), so only the kept rows are fetched; diagnosis photos for all rows are
// loaded in a single batched query to avoid an N+1.
func (s *PatientService) buildHistoryEntries(ctx context.Context, patientID uint, from, to string) ([]dto.PatientHistoryEntry, error) {
	db := s.scheduleRepo.DB().WithContext(ctx)
	type joined struct {
		SPID          uint
		ScheduleID    uint
		Date          string
		SPStart       string
		SPEnd         string
		Location      string
		Status        string
		NoShowReason  string
		PrepaidAmount int
		ActualAmount  int
		TName         string
	}
	q := db.Table("schedule_patients as sp").
		Select(`sp.id as sp_id, sp.schedule_id, sp.start_time as sp_start, sp.end_time as sp_end,
			sp.status, sp.no_show_reason, sp.prepaid_amount, sp.actual_amount,
			schedules.date, schedules.location, users.name as t_name`).
		Joins("JOIN schedules ON schedules.id = sp.schedule_id").
		Joins("JOIN users ON users.id = schedules.translator_id").
		Where("sp.patient_id = ?", patientID)
	if from != "" {
		q = q.Where("schedules.date >= ?", from)
	}
	if to != "" {
		if toExcl, ok := nextDay(to); ok {
			q = q.Where("schedules.date < ?", toExcl)
		}
	}
	var rows []joined
	if err := q.Order("schedules.date DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}

	// Batch-load photos for all rows in one query, then group by slot id.
	spIDs := make([]uint, len(rows))
	for i, r := range rows {
		spIDs[i] = r.SPID
	}
	photos, err := s.photoRepo.WithCtx(ctx).FindBySchedulePatientIDs(spIDs)
	if err != nil {
		return nil, err
	}
	photosBySP := make(map[uint][]string, len(spIDs))
	for _, p := range photos {
		photosBySP[p.SchedulePatientID] = append(photosBySP[p.SchedulePatientID], p.PhotoURL)
	}

	entries := make([]dto.PatientHistoryEntry, 0, len(rows))
	for _, r := range rows {
		photoURLs := photosBySP[r.SPID]
		if photoURLs == nil {
			photoURLs = []string{}
		}
		// sqlite returns date as RFC3339; postgres returns YYYY-MM-DD. Trim T... if present.
		dateOnly := r.Date
		if idx := indexT(dateOnly); idx > 0 {
			dateOnly = dateOnly[:idx]
		}
		entries = append(entries, dto.PatientHistoryEntry{
			ScheduleID:      r.ScheduleID,
			Date:            dateOnly,
			StartTime:       r.SPStart,
			EndTime:         r.SPEnd,
			Location:        r.Location,
			TranslatorName:  r.TName,
			Status:          r.Status,
			NoShowReason:    r.NoShowReason,
			DiagnosisPhotos: photoURLs,
			PrepaidAmount:   r.PrepaidAmount,
			ActualAmount:    r.ActualAmount,
		})
	}
	return entries, nil
}

// indexT returns the index of 'T' in s or -1 if absent.
func indexT(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == 'T' {
			return i
		}
	}
	return -1
}

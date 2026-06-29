package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Sentinel errors returned by ScheduleService.
var (
	ErrScheduleNotFound      = errors.New("schedule not found")
	ErrInvalidDateFormat     = errors.New("invalid date format, use YYYY-MM-DD")
	ErrRecurrenceUntilReq    = errors.New("recurrenceUntil is required when recurrenceRule is set")
	ErrRecurrenceBeforeStart = errors.New("recurrenceUntil must be after or equal to date")
	ErrInvalidRecurrence     = errors.New("invalid recurrenceRule")
	ErrNoDatesGenerated      = errors.New("no dates generated for the given recurrence rule and range")

	// Stage 3 multi-patient sentinels.
	ErrSchedulePatientsRequired   = errors.New("schedule must contain at least one patient")
	ErrDuplicatePatientInSchedule = errors.New("the same patient cannot appear twice in one schedule")
	ErrPatientTimeOutOfRange      = errors.New("patient time slot is outside the schedule's overall start/end")
	ErrPatientEndBeforeStart      = errors.New("patient end_time must be after start_time")
)

// ScheduleService handles schedule management business logic.
type ScheduleService struct {
	scheduleRepo *repository.ScheduleRepository
	checkinRepo  *repository.CheckinRepository
	userRepo     *repository.UserRepository
	// Stage 3 dependencies — optional so old tests that use the 3-arg
	// constructor still work (they don't exercise multi-patient flows).
	spRepo      *repository.SchedulePatientRepository
	patientRepo *repository.PatientRepository
}

// NewScheduleService creates a new ScheduleService with legacy 3-repo signature.
// Multi-patient flows require WithPatientRepos to be called.
func NewScheduleService(
	scheduleRepo *repository.ScheduleRepository,
	checkinRepo *repository.CheckinRepository,
	userRepo *repository.UserRepository,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		checkinRepo:  checkinRepo,
		userRepo:     userRepo,
	}
}

// WithPatientRepos wires up the SchedulePatient + Patient repos required by
// stage-3 multi-patient features. Returns the service for chaining.
func (s *ScheduleService) WithPatientRepos(
	spRepo *repository.SchedulePatientRepository,
	patientRepo *repository.PatientRepository,
) *ScheduleService {
	s.spRepo = spRepo
	s.patientRepo = patientRepo
	return s
}

// DefaultRecentScheduleLimit caps the unfiltered admin list ("latest created
// schedules") view.
const DefaultRecentScheduleLimit = 100

// List returns schedules with optional filters and checkin status.
//
// When no filter is supplied (no translator, date range or location) it returns
// the most recently created schedules (created_at DESC, capped at
// DefaultRecentScheduleLimit) — the default admin view. Any filter switches to
// the full filtered query ordered by date ASC.
func (s *ScheduleService) List(ctx context.Context, translatorID uint, dateFrom, dateTo, location string, page, pageSize int) ([]dto.ScheduleResponse, int64, error) {
	var (
		schedules []model.Schedule
		total     int64
		err       error
	)
	if translatorID == 0 && dateFrom == "" && dateTo == "" && location == "" {
		schedules, total, err = s.scheduleRepo.WithCtx(ctx).FindRecentByCreated(page, pageSize)
	} else {
		schedules, total, err = s.scheduleRepo.WithCtx(ctx).FindAll(translatorID, dateFrom, dateTo, location, page, pageSize)
	}
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.toResponseList(ctx, schedules)
	if err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

// ListForTranslator returns schedules for a specific translator.
func (s *ScheduleService) ListForTranslator(ctx context.Context, translatorID uint, dateFrom, dateTo string) ([]dto.ScheduleResponse, error) {
	schedules, err := s.scheduleRepo.WithCtx(ctx).FindByTranslator(translatorID, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	return s.toResponseList(ctx, schedules)
}

// Create adds a new schedule entry (or multiple if RecurrenceRule is set).
//
// Stage 3 flow: when req.Patients is non-empty the service validates the
// payload and creates schedule + schedule_patients rows in a single transaction.
// Legacy single-patient path (PatientName only) is preserved for backward
// compat with stage 1/2 callers and tests.
func (s *ScheduleService) Create(ctx context.Context, req dto.CreateScheduleRequest) (*dto.ScheduleResponse, error) {
	user, err := s.userRepo.WithCtx(ctx).FindByID(req.TranslatorID)
	if err != nil {
		return nil, ErrTranslatorNotFound
	}
	if user.Role != "translator" {
		return nil, ErrNotATranslator
	}

	if req.RecurrenceRule != "" {
		return s.createRecurring(ctx, req)
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, ErrInvalidDateFormat
	}

	// Multi-patient mode: validate then create in a transaction.
	if len(req.Patients) > 0 {
		if err := s.validateSchedulePatients(ctx, req.StartTime, req.EndTime, req.Patients); err != nil {
			return nil, err
		}
		return s.createWithPatients(ctx, req, date)
	}

	// Stage-1 backward compat: explicit empty Patients[] with no PatientName → error.
	if req.Patients != nil && len(req.Patients) == 0 && req.PatientName == "" {
		return nil, ErrSchedulePatientsRequired
	}

	schedule := &model.Schedule{
		TranslatorID: req.TranslatorID,
		Date:         date,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		Location:     req.Location,
		PatientName:  optionalString(req.PatientName),
		Note:         req.Note,
	}

	schRepo := s.scheduleRepo.WithCtx(ctx)
	if err := schRepo.Create(schedule); err != nil {
		return nil, err
	}
	schedule, err = schRepo.FindByID(schedule.ID)
	if err != nil {
		return nil, err
	}
	resp := s.toResponse(schedule, "none")
	return &resp, nil
}

// createWithPatients persists a schedule together with its SchedulePatient
// rows in one transaction. Validation must have already run.
func (s *ScheduleService) createWithPatients(ctx context.Context, req dto.CreateScheduleRequest, date time.Time) (*dto.ScheduleResponse, error) {
	schedule := &model.Schedule{
		TranslatorID: req.TranslatorID,
		Date:         date,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		Location:     req.Location,
		Note:         req.Note,
	}

	db := s.scheduleRepo.DB().WithContext(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(schedule).Error; err != nil {
			return err
		}
		rows := make([]*model.SchedulePatient, 0, len(req.Patients))
		for i, p := range req.Patients {
			rows = append(rows, &model.SchedulePatient{
				ScheduleID:    schedule.ID,
				PatientID:     p.PatientID,
				StartTime:     p.StartTime,
				EndTime:       p.EndTime,
				OrderIdx:      i,
				Status:        model.SchedulePatientStatusPending,
				PrepaidAmount: p.PrepaidAmount,
			})
		}
		return tx.Create(&rows).Error
	})
	if err != nil {
		return nil, err
	}

	reloaded, err := s.scheduleRepo.WithCtx(ctx).FindByID(schedule.ID)
	if err != nil {
		return nil, err
	}
	resp := s.toResponse(reloaded, "none")
	return &resp, nil
}

// validateSchedulePatients runs business validation on a Patients payload:
//   - patient end > start
//   - patient slot within overall schedule start/end
//   - no duplicate patient IDs within one schedule
//   - every patient ID resolves to an existing Patient row
func (s *ScheduleService) validateSchedulePatients(ctx context.Context, overallStart, overallEnd string, patients []dto.SchedulePatientPayload) error {
	if len(patients) == 0 {
		return ErrSchedulePatientsRequired
	}
	seen := map[uint]bool{}
	for _, p := range patients {
		if p.EndTime <= p.StartTime {
			return ErrPatientEndBeforeStart
		}
		if p.StartTime < overallStart || p.EndTime > overallEnd {
			return ErrPatientTimeOutOfRange
		}
		if seen[p.PatientID] {
			return ErrDuplicatePatientInSchedule
		}
		seen[p.PatientID] = true
		if s.patientRepo != nil {
			if _, err := s.patientRepo.WithCtx(ctx).FindByID(p.PatientID); err != nil {
				return ErrPatientNotFound
			}
		}
	}
	return nil
}

// createRecurring creates multiple schedule records based on a recurrence rule.
func (s *ScheduleService) createRecurring(ctx context.Context, req dto.CreateScheduleRequest) (*dto.ScheduleResponse, error) {
	if req.RecurrenceUntil == "" {
		return nil, ErrRecurrenceUntilReq
	}

	startDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, ErrInvalidDateFormat
	}

	untilDate, err := time.Parse("2006-01-02", req.RecurrenceUntil)
	if err != nil {
		return nil, ErrInvalidDateFormat
	}

	if untilDate.Before(startDate) {
		return nil, ErrRecurrenceBeforeStart
	}

	rule := req.RecurrenceRule
	groupID := uuid.New().String()

	dates, err := expandRecurrenceDates(startDate, untilDate, rule)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRecurrence, err)
	}

	if len(dates) == 0 {
		return nil, ErrNoDatesGenerated
	}

	schedules := make([]*model.Schedule, 0, len(dates))
	for _, d := range dates {
		d := d // capture range variable
		schedules = append(schedules, &model.Schedule{
			TranslatorID:      req.TranslatorID,
			Date:              d,
			StartTime:         req.StartTime,
			EndTime:           req.EndTime,
			Location:          req.Location,
			PatientName:       optionalString(req.PatientName),
			Note:              req.Note,
			RecurrenceRule:    &rule,
			RecurrenceGroupID: &groupID,
		})
	}

	schRepo := s.scheduleRepo.WithCtx(ctx)
	if err := schRepo.CreateBatch(schedules); err != nil {
		return nil, err
	}

	// Reload first schedule with translator
	first, err := schRepo.FindByID(schedules[0].ID)
	if err != nil {
		return nil, err
	}

	resp := s.toResponse(first, "none")
	return &resp, nil
}

// expandRecurrenceDates returns all dates between start and until (inclusive) matching the rule.
func expandRecurrenceDates(start, until time.Time, rule string) ([]time.Time, error) {
	var dates []time.Time

	// Normalize to midnight
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	until = time.Date(until.Year(), until.Month(), until.Day(), 0, 0, 0, 0, until.Location())

	switch {
	case rule == "daily":
		for d := start; !d.After(until); d = d.AddDate(0, 0, 1) {
			dates = append(dates, d)
		}

	case strings.HasPrefix(rule, "weekly:"):
		parts := strings.TrimPrefix(rule, "weekly:")
		weekdays, err := parseIntList(parts)
		if err != nil {
			return nil, fmt.Errorf("weekly rule must be like 'weekly:1,3,5': %w", err)
		}
		wdSet := make(map[int]bool)
		for _, w := range weekdays {
			if w < 0 || w > 6 {
				return nil, errors.New("weekday values must be 0-6 (0=Sunday)")
			}
			wdSet[w] = true
		}
		for d := start; !d.After(until); d = d.AddDate(0, 0, 1) {
			if wdSet[int(d.Weekday())] {
				dates = append(dates, d)
			}
		}

	case strings.HasPrefix(rule, "monthly:"):
		parts := strings.TrimPrefix(rule, "monthly:")
		days, err := parseIntList(parts)
		if err != nil {
			return nil, fmt.Errorf("monthly rule must be like 'monthly:5,20': %w", err)
		}
		for _, day := range days {
			if day < 1 || day > 31 {
				return nil, errors.New("monthly day values must be 1-31")
			}
		}
		// Walk month by month. For each target day, clamp to the last day of
		// the month so day=31 behaves as "last day" in shorter months (e.g.
		// February yields 28/29, April yields 30).
		cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
		endMonth := time.Date(until.Year(), until.Month(), 1, 0, 0, 0, 0, until.Location())
		seen := make(map[string]bool)
		for !cur.After(endMonth) {
			year, month := cur.Year(), cur.Month()
			lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, cur.Location()).Day()
			for _, target := range days {
				day := target
				if day > lastDay {
					day = lastDay
				}
				candidate := time.Date(year, month, day, 0, 0, 0, 0, cur.Location())
				if candidate.Before(start) || candidate.After(until) {
					continue
				}
				key := candidate.Format("2006-01-02")
				if !seen[key] {
					seen[key] = true
					dates = append(dates, candidate)
				}
			}
			cur = cur.AddDate(0, 1, 0)
		}

	default:
		return nil, fmt.Errorf("unknown rule %q, supported: daily, weekly:N,..., monthly:N,...", rule)
	}

	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates, nil
}

// parseIntList parses a comma-separated string of integers.
func parseIntList(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", p)
		}
		result = append(result, n)
	}
	return result, nil
}

// Update modifies an existing schedule.
func (s *ScheduleService) Update(ctx context.Context, id uint, req dto.UpdateScheduleRequest) (*dto.ScheduleResponse, error) {
	schedule, err := s.scheduleRepo.WithCtx(ctx).FindByID(id)
	if err != nil {
		return nil, ErrScheduleNotFound
	}

	if req.Date != nil {
		date, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return nil, ErrInvalidDateFormat
		}
		schedule.Date = date
	}
	if req.StartTime != nil {
		schedule.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		schedule.EndTime = *req.EndTime
	}
	if req.Location != nil {
		schedule.Location = *req.Location
	}
	if req.PatientName != nil {
		schedule.PatientName = req.PatientName
	}
	if req.Note != nil {
		schedule.Note = *req.Note
	}

	// Stage 3: if Patients was supplied, validate then replace the whole list
	// inside a transaction together with the schedule update.
	if req.Patients != nil {
		if err := s.validateSchedulePatients(ctx, schedule.StartTime, schedule.EndTime, *req.Patients); err != nil {
			return nil, err
		}
		db := s.scheduleRepo.DB().WithContext(ctx)
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(schedule).Error; err != nil {
				return err
			}
			if err := tx.Where("schedule_id = ?", schedule.ID).Delete(&model.SchedulePatient{}).Error; err != nil {
				return err
			}
			rows := make([]*model.SchedulePatient, 0, len(*req.Patients))
			for i, p := range *req.Patients {
				rows = append(rows, &model.SchedulePatient{
					ScheduleID:    schedule.ID,
					PatientID:     p.PatientID,
					StartTime:     p.StartTime,
					EndTime:       p.EndTime,
					OrderIdx:      i,
					Status:        model.SchedulePatientStatusPending,
					PrepaidAmount: p.PrepaidAmount,
				})
			}
			return tx.Create(&rows).Error
		})
		if err != nil {
			return nil, err
		}
	} else {
		schRepo := s.scheduleRepo.WithCtx(ctx)
		if err := schRepo.Update(schedule); err != nil {
			return nil, err
		}
	}

	// Reload
	schedule, err = s.scheduleRepo.WithCtx(ctx).FindByID(schedule.ID)
	if err != nil {
		return nil, err
	}

	status := s.getCheckinStatus(ctx, schedule.ID)
	resp := s.toResponse(schedule, status)
	return &resp, nil
}

// Delete removes a schedule by ID.
// Associated checkins are deleted first to satisfy the FK constraint.
func (s *ScheduleService) Delete(ctx context.Context, id uint) error {
	repo := s.scheduleRepo.WithCtx(ctx)
	_, err := repo.FindByID(id)
	if err != nil {
		return ErrScheduleNotFound
	}
	if err := s.checkinRepo.WithCtx(ctx).DeleteByScheduleID(id); err != nil {
		return err
	}
	// Stage 3+: cascade-delete schedule_patients before removing the schedule
	// row, otherwise the FK fk_schedules_patients blocks the delete.
	if s.spRepo != nil {
		if err := s.spRepo.WithCtx(ctx).DeleteByScheduleID(id); err != nil {
			return err
		}
	}
	return repo.Delete(id)
}

// DeleteRecurrenceGroup removes every schedule sharing the same
// recurrence_group_id as the given schedule. If the schedule isn't part of a
// group, it falls back to deleting just that single record.
// Associated checkins are deleted first to satisfy the FK constraint.
func (s *ScheduleService) DeleteRecurrenceGroup(ctx context.Context, id uint) (int64, error) {
	repo := s.scheduleRepo.WithCtx(ctx)
	schedule, err := repo.FindByID(id)
	if err != nil {
		return 0, ErrScheduleNotFound
	}
	if schedule.RecurrenceGroupID == nil || *schedule.RecurrenceGroupID == "" {
		if err := s.checkinRepo.WithCtx(ctx).DeleteByScheduleID(id); err != nil {
			return 0, err
		}
		if s.spRepo != nil {
			if err := s.spRepo.WithCtx(ctx).DeleteByScheduleID(id); err != nil {
				return 0, err
			}
		}
		if err := repo.Delete(id); err != nil {
			return 0, err
		}
		return 1, nil
	}
	// Bulk group delete: collect schedule IDs first, then cascade checkins +
	// schedule_patients, then schedules.
	scheduleIDs, err := repo.IDsByRecurrenceGroup(*schedule.RecurrenceGroupID)
	if err != nil {
		return 0, err
	}
	if err := s.checkinRepo.WithCtx(ctx).DeleteByScheduleIDs(scheduleIDs); err != nil {
		return 0, err
	}
	if s.spRepo != nil {
		if err := s.spRepo.WithCtx(ctx).DeleteByScheduleIDs(scheduleIDs); err != nil {
			return 0, err
		}
	}
	return repo.DeleteByRecurrenceGroup(*schedule.RecurrenceGroupID)
}

// getCheckinStatus determines the checkin status for a schedule.
func (s *ScheduleService) getCheckinStatus(ctx context.Context, scheduleID uint) string {
	checkins, err := s.checkinRepo.WithCtx(ctx).FindByScheduleID(scheduleID)
	if err != nil || len(checkins) == 0 {
		return "none"
	}

	hasArrive := false
	hasLeave := false
	hasMakeup := false

	for _, c := range checkins {
		if c.IsMakeup {
			hasMakeup = true
		}
		if c.Type == "arrive" {
			hasArrive = true
		}
		if c.Type == "leave" {
			hasLeave = true
		}
	}

	// "completed" takes priority: both arrive and leave are recorded,
	// regardless of whether they were makeup or on-time.
	if hasArrive && hasLeave {
		return "completed"
	}
	// Makeup arrive done but leave still pending.
	if hasMakeup {
		return "makeup"
	}
	if hasArrive {
		return "arrived"
	}
	return "none"
}

func (s *ScheduleService) toResponse(schedule *model.Schedule, checkinStatus string) dto.ScheduleResponse {
	pn := ""
	if schedule.PatientName != nil {
		pn = *schedule.PatientName
	}
	patients := make([]dto.SchedulePatientResponse, 0, len(schedule.Patients))
	for _, sp := range schedule.Patients {
		patients = append(patients, dto.SchedulePatientResponse{
			ID:           sp.ID,
			PatientID:    sp.PatientID,
			PatientName:  sp.Patient.Name,
			PatientPhone: sp.Patient.Phone,
			IDType:       sp.Patient.IDType,
			IDNumber:     sp.Patient.IDNumber,
			StartTime:     sp.StartTime,
			EndTime:       sp.EndTime,
			Status:        sp.Status,
			NoShowReason:  sp.NoShowReason,
			PrepaidAmount: sp.PrepaidAmount,
			ActualAmount:  sp.ActualAmount,
		})
	}
	return dto.ScheduleResponse{
		ID:                schedule.ID,
		TranslatorID:      schedule.TranslatorID,
		TranslatorName:    schedule.Translator.Name,
		Date:              schedule.Date.Format("2006-01-02"),
		StartTime:         schedule.StartTime,
		EndTime:           schedule.EndTime,
		Location:          schedule.Location,
		PatientName:       pn,
		Note:              schedule.Note,
		CheckinStatus:     checkinStatus,
		RecurrenceGroupID: schedule.RecurrenceGroupID,
		Patients:          patients,
	}
}

func (s *ScheduleService) toResponseList(ctx context.Context, schedules []model.Schedule) ([]dto.ScheduleResponse, error) {
	result := make([]dto.ScheduleResponse, len(schedules))
	for i, sch := range schedules {
		status := s.getCheckinStatus(ctx, sch.ID)
		result[i] = s.toResponse(&sch, status)
	}
	return result, nil
}

// ScheduleImportRow describes one row of an uploaded schedule spreadsheet.
type ScheduleImportRow struct {
	RowNumber    int
	TranslatorID uint
	Date         string
	StartTime    string
	EndTime      string
	Location     string
	PatientName  string
	Note         string
	Error        string
}

// BatchImportSchedules creates schedules for each valid row. Rows are persisted
// individually so a single bad row doesn't abort the import. The returned
// counts and per-row errors let callers surface a meaningful report.
func (s *ScheduleService) BatchImportSchedules(ctx context.Context, rows []ScheduleImportRow) (success int, failed []ScheduleImportRow) {
	userRepo := s.userRepo.WithCtx(ctx)
	schRepo := s.scheduleRepo.WithCtx(ctx)
	for _, r := range rows {
		if r.Error != "" {
			failed = append(failed, r)
			continue
		}
		date, err := time.Parse("2006-01-02", r.Date)
		if err != nil {
			r.Error = "invalid date format"
			failed = append(failed, r)
			continue
		}
		user, err := userRepo.FindByID(r.TranslatorID)
		if err != nil || user.Role != "translator" {
			r.Error = "translator not found"
			failed = append(failed, r)
			continue
		}
		schedule := &model.Schedule{
			TranslatorID: r.TranslatorID,
			Date:         date,
			StartTime:    r.StartTime,
			EndTime:      r.EndTime,
			Location:     r.Location,
			PatientName:  optionalString(r.PatientName),
			Note:         r.Note,
		}
		if err := schRepo.Create(schedule); err != nil {
			r.Error = err.Error()
			failed = append(failed, r)
			continue
		}
		success++
	}
	return success, failed
}

// Unexported but used to check gorm import usage
var _ = gorm.ErrRecordNotFound

// optionalString returns nil if s is empty, otherwise a pointer to s.
// Used when assigning to nullable string columns.
func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ScheduleImportRowV2 is one row of the stage-3 flat Excel format.
//
//	A=Code | B=TranslatorID | C=Date | D=OverallStart | E=OverallEnd |
//	F=Location | G=PatientID | H=PatientStart | I=PatientEnd | J=Note(optional)
//
// Rows sharing the same Code merge into one schedule with multiple patients.
type ScheduleImportRowV2 struct {
	RowNumber    int
	Code         string
	TranslatorID uint
	Date         string
	OverallStart string
	OverallEnd   string
	Location     string
	PatientID    uint
	PatientStart string
	PatientEnd   string
	Note         string
	Error        string
}

// ScheduleImportResultV2 aggregates the outcome of BatchImportSchedulesV2.
type ScheduleImportResultV2 struct {
	SuccessSchedules int                   `json:"successSchedules"`
	SuccessPatients  int                   `json:"successPatients"`
	Failed           []ScheduleImportRowV2 `json:"failed"`
}

// BatchImportSchedulesV2 groups input rows by Code and creates one schedule
// per code with all matching patients. Failed groups are surfaced in Failed;
// other groups still succeed.
func (s *ScheduleService) BatchImportSchedulesV2(ctx context.Context, rows []ScheduleImportRowV2) (*ScheduleImportResultV2, error) {
	result := &ScheduleImportResultV2{}
	if len(rows) == 0 {
		return result, nil
	}

	// Group by code while preserving first-seen order.
	groupOrder := []string{}
	groups := map[string][]ScheduleImportRowV2{}
	for _, r := range rows {
		if _, ok := groups[r.Code]; !ok {
			groupOrder = append(groupOrder, r.Code)
		}
		groups[r.Code] = append(groups[r.Code], r)
	}

	for _, code := range groupOrder {
		group := groups[code]
		if code == "" {
			for _, r := range group {
				r.Error = "schedule code is required"
				result.Failed = append(result.Failed, r)
			}
			continue
		}
		s.importGroup(ctx, group, result)
	}
	return result, nil
}

func (s *ScheduleService) importGroup(ctx context.Context, group []ScheduleImportRowV2, result *ScheduleImportResultV2) {
	// Verify meta consistency across rows of the same code.
	head := group[0]
	for _, r := range group[1:] {
		if r.TranslatorID != head.TranslatorID || r.Date != head.Date ||
			r.OverallStart != head.OverallStart || r.OverallEnd != head.OverallEnd ||
			r.Location != head.Location {
			r.Error = "conflicting schedule meta for code " + head.Code
			result.Failed = append(result.Failed, r)
			return
		}
	}

	// Build CreateScheduleRequest and reuse Create() so all validation rules
	// (time-in-range, duplicate patient, patient exists, translator exists,
	// date format) stay in one place.
	patients := make([]dto.SchedulePatientPayload, 0, len(group))
	for _, r := range group {
		patients = append(patients, dto.SchedulePatientPayload{
			PatientID: r.PatientID,
			StartTime: r.PatientStart,
			EndTime:   r.PatientEnd,
		})
	}
	req := dto.CreateScheduleRequest{
		TranslatorID: head.TranslatorID,
		Date:         head.Date,
		StartTime:    head.OverallStart,
		EndTime:      head.OverallEnd,
		Location:     head.Location,
		Patients:     patients,
		Note:         head.Note,
	}
	if _, err := s.Create(ctx, req); err != nil {
		failedRow := group[0]
		failedRow.Error = err.Error()
		result.Failed = append(result.Failed, failedRow)
		return
	}
	result.SuccessSchedules++
	result.SuccessPatients += len(group)
}

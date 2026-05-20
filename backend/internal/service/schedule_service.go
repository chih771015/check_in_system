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
)

// ScheduleService handles schedule management business logic.
type ScheduleService struct {
	scheduleRepo *repository.ScheduleRepository
	checkinRepo  *repository.CheckinRepository
	userRepo     *repository.UserRepository
}

// NewScheduleService creates a new ScheduleService.
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

// List returns schedules with optional filters and checkin status.
func (s *ScheduleService) List(ctx context.Context, translatorID uint, dateFrom, dateTo, location string) ([]dto.ScheduleResponse, error) {
	schedules, err := s.scheduleRepo.WithCtx(ctx).FindAll(translatorID, dateFrom, dateTo, location)
	if err != nil {
		return nil, err
	}
	return s.toResponseList(ctx, schedules)
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
func (s *ScheduleService) Create(ctx context.Context, req dto.CreateScheduleRequest) (*dto.ScheduleResponse, error) {
	// Verify translator exists
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

	// Reload with translator
	schedule, err = schRepo.FindByID(schedule.ID)
	if err != nil {
		return nil, err
	}

	resp := s.toResponse(schedule, "none")
	return &resp, nil
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

	schRepo := s.scheduleRepo.WithCtx(ctx)
	if err := schRepo.Update(schedule); err != nil {
		return nil, err
	}

	// Reload
	schedule, err = schRepo.FindByID(schedule.ID)
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
		if err := repo.Delete(id); err != nil {
			return 0, err
		}
		return 1, nil
	}
	// Bulk group delete: collect schedule IDs first, then delete checkins, then schedules.
	scheduleIDs, err := repo.IDsByRecurrenceGroup(*schedule.RecurrenceGroupID)
	if err != nil {
		return 0, err
	}
	if err := s.checkinRepo.WithCtx(ctx).DeleteByScheduleIDs(scheduleIDs); err != nil {
		return 0, err
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
			StartTime:    sp.StartTime,
			EndTime:      sp.EndTime,
			Status:       sp.Status,
			NoShowReason: sp.NoShowReason,
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

package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
func (s *ScheduleService) List(translatorID uint, dateFrom, dateTo, location string) ([]dto.ScheduleResponse, error) {
	schedules, err := s.scheduleRepo.FindAll(translatorID, dateFrom, dateTo, location)
	if err != nil {
		return nil, err
	}
	return s.toResponseList(schedules)
}

// ListForTranslator returns schedules for a specific translator.
func (s *ScheduleService) ListForTranslator(translatorID uint, dateFrom, dateTo string) ([]dto.ScheduleResponse, error) {
	schedules, err := s.scheduleRepo.FindByTranslator(translatorID, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	return s.toResponseList(schedules)
}

// Create adds a new schedule entry (or multiple if RecurrenceRule is set).
func (s *ScheduleService) Create(req dto.CreateScheduleRequest) (*dto.ScheduleResponse, error) {
	// Verify translator exists
	user, err := s.userRepo.FindByID(req.TranslatorID)
	if err != nil {
		return nil, errors.New("translator not found")
	}
	if user.Role != "translator" {
		return nil, errors.New("user is not a translator")
	}

	if req.RecurrenceRule != "" {
		return s.createRecurring(req)
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, errors.New("invalid date format, use YYYY-MM-DD")
	}

	schedule := &model.Schedule{
		TranslatorID: req.TranslatorID,
		Date:         date,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		Location:     req.Location,
		PatientName:  req.PatientName,
		Note:         req.Note,
	}

	if err := s.scheduleRepo.Create(schedule); err != nil {
		return nil, err
	}

	// Reload with translator
	schedule, err = s.scheduleRepo.FindByID(schedule.ID)
	if err != nil {
		return nil, err
	}

	resp := s.toResponse(schedule, "none")
	return &resp, nil
}

// createRecurring creates multiple schedule records based on a recurrence rule.
func (s *ScheduleService) createRecurring(req dto.CreateScheduleRequest) (*dto.ScheduleResponse, error) {
	if req.RecurrenceUntil == "" {
		return nil, errors.New("recurrenceUntil is required when recurrenceRule is set")
	}

	startDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, errors.New("invalid date format, use YYYY-MM-DD")
	}

	untilDate, err := time.Parse("2006-01-02", req.RecurrenceUntil)
	if err != nil {
		return nil, errors.New("invalid recurrenceUntil format, use YYYY-MM-DD")
	}

	if untilDate.Before(startDate) {
		return nil, errors.New("recurrenceUntil must be after or equal to date")
	}

	rule := req.RecurrenceRule
	groupID := uuid.New().String()

	dates, err := expandRecurrenceDates(startDate, untilDate, rule)
	if err != nil {
		return nil, fmt.Errorf("invalid recurrenceRule: %w", err)
	}

	if len(dates) == 0 {
		return nil, errors.New("no dates generated for the given recurrence rule and range")
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
			PatientName:       req.PatientName,
			Note:              req.Note,
			RecurrenceRule:    &rule,
			RecurrenceGroupID: &groupID,
		})
	}

	if err := s.scheduleRepo.CreateBatch(schedules); err != nil {
		return nil, err
	}

	// Reload first schedule with translator
	first, err := s.scheduleRepo.FindByID(schedules[0].ID)
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
		daySet := make(map[int]bool)
		for _, day := range days {
			if day < 1 || day > 31 {
				return nil, errors.New("monthly day values must be 1-31")
			}
			daySet[day] = true
		}
		for d := start; !d.After(until); d = d.AddDate(0, 0, 1) {
			if daySet[d.Day()] {
				dates = append(dates, d)
			}
		}

	default:
		return nil, fmt.Errorf("unknown rule %q, supported: daily, weekly:N,..., monthly:N,...", rule)
	}

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
func (s *ScheduleService) Update(id uint, req dto.UpdateScheduleRequest) (*dto.ScheduleResponse, error) {
	schedule, err := s.scheduleRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("schedule not found")
	}

	if req.Date != nil {
		date, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return nil, errors.New("invalid date format, use YYYY-MM-DD")
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
		schedule.PatientName = *req.PatientName
	}
	if req.Note != nil {
		schedule.Note = *req.Note
	}

	if err := s.scheduleRepo.Update(schedule); err != nil {
		return nil, err
	}

	// Reload
	schedule, err = s.scheduleRepo.FindByID(schedule.ID)
	if err != nil {
		return nil, err
	}

	status := s.getCheckinStatus(schedule.ID)
	resp := s.toResponse(schedule, status)
	return &resp, nil
}

// Delete removes a schedule by ID.
func (s *ScheduleService) Delete(id uint) error {
	_, err := s.scheduleRepo.FindByID(id)
	if err != nil {
		return errors.New("schedule not found")
	}
	return s.scheduleRepo.Delete(id)
}

// getCheckinStatus determines the checkin status for a schedule.
func (s *ScheduleService) getCheckinStatus(scheduleID uint) string {
	checkins, err := s.checkinRepo.FindByScheduleID(scheduleID)
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

	if hasMakeup {
		return "makeup"
	}
	if hasArrive && hasLeave {
		return "completed"
	}
	if hasArrive {
		return "arrived"
	}
	return "none"
}

func (s *ScheduleService) toResponse(schedule *model.Schedule, checkinStatus string) dto.ScheduleResponse {
	return dto.ScheduleResponse{
		ID:             schedule.ID,
		TranslatorID:   schedule.TranslatorID,
		TranslatorName: schedule.Translator.Name,
		Date:           schedule.Date.Format("2006-01-02"),
		StartTime:      schedule.StartTime,
		EndTime:        schedule.EndTime,
		Location:       schedule.Location,
		PatientName:    schedule.PatientName,
		Note:           schedule.Note,
		CheckinStatus:  checkinStatus,
	}
}

func (s *ScheduleService) toResponseList(schedules []model.Schedule) ([]dto.ScheduleResponse, error) {
	result := make([]dto.ScheduleResponse, len(schedules))
	for i, sch := range schedules {
		status := s.getCheckinStatus(sch.ID)
		result[i] = s.toResponse(&sch, status)
	}
	return result, nil
}

// Unexported but used to check gorm import usage
var _ = gorm.ErrRecordNotFound

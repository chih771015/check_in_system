package service

import (
	"errors"
	"time"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

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

// Create adds a new schedule entry.
func (s *ScheduleService) Create(req dto.CreateScheduleRequest) (*dto.ScheduleResponse, error) {
	// Verify translator exists
	user, err := s.userRepo.FindByID(req.TranslatorID)
	if err != nil {
		return nil, errors.New("translator not found")
	}
	if user.Role != "translator" {
		return nil, errors.New("user is not a translator")
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

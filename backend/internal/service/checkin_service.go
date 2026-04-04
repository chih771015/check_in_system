package service

import (
	"errors"
	"time"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"gorm.io/gorm"
)

// CheckinService handles check-in business logic.
type CheckinService struct {
	checkinRepo  *repository.CheckinRepository
	scheduleRepo *repository.ScheduleRepository
	userRepo     *repository.UserRepository
}

// NewCheckinService creates a new CheckinService.
func NewCheckinService(
	checkinRepo *repository.CheckinRepository,
	scheduleRepo *repository.ScheduleRepository,
	userRepo *repository.UserRepository,
) *CheckinService {
	return &CheckinService{
		checkinRepo:  checkinRepo,
		scheduleRepo: scheduleRepo,
		userRepo:     userRepo,
	}
}

// Checkin processes a translator's check-in (arrive or leave).
func (s *CheckinService) Checkin(
	translatorID uint,
	scheduleID uint,
	checkinType string,
	lat, lng float64,
	address, selfieURL, envURL string,
	isMakeup bool,
	makeupReason string,
) (*dto.CheckinResponse, error) {
	// Validate schedule exists and belongs to translator
	schedule, err := s.scheduleRepo.FindByID(scheduleID)
	if err != nil {
		return nil, errors.New("schedule not found")
	}
	if schedule.TranslatorID != translatorID {
		return nil, errors.New("schedule does not belong to this translator")
	}

	// Check for duplicate checkin type
	existing, err := s.checkinRepo.FindByScheduleAndType(scheduleID, checkinType)
	if err == nil && existing != nil {
		return nil, errors.New("already checked in with type: " + checkinType)
	}

	// If leaving, ensure arrival was recorded first
	if checkinType == "leave" {
		_, err := s.checkinRepo.FindByScheduleAndType(scheduleID, "arrive")
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("must check in (arrive) before checking out (leave)")
			}
			return nil, errors.New("failed to verify arrival status")
		}
	}

	// Get translator info
	user, err := s.userRepo.FindByID(translatorID)
	if err != nil {
		return nil, errors.New("translator not found")
	}

	checkin := &model.Checkin{
		ScheduleID:     scheduleID,
		TranslatorID:   translatorID,
		Type:           checkinType,
		CheckinTime:    time.Now(),
		Latitude:       lat,
		Longitude:      lng,
		Address:        address,
		SelfieURL:      selfieURL,
		EnvironmentURL: envURL,
		IsMakeup:       isMakeup,
		MakeupReason:   makeupReason,
	}

	if err := s.checkinRepo.Create(checkin); err != nil {
		return nil, errors.New("failed to create checkin record")
	}

	return &dto.CheckinResponse{
		ID:             checkin.ID,
		ScheduleID:     checkin.ScheduleID,
		TranslatorID:   checkin.TranslatorID,
		TranslatorName: user.Name,
		Type:           checkin.Type,
		CheckinTime:    checkin.CheckinTime,
		Latitude:       checkin.Latitude,
		Longitude:      checkin.Longitude,
		Address:        checkin.Address,
		SelfieURL:      checkin.SelfieURL,
		EnvironmentURL: checkin.EnvironmentURL,
		IsMakeup:       checkin.IsMakeup,
		MakeupReason:   checkin.MakeupReason,
		CreatedAt:      checkin.CreatedAt,
	}, nil
}

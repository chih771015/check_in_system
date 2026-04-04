package repository

import (
	"translator-checkin/internal/model"

	"gorm.io/gorm"
)

// CheckinRepository handles database operations for checkins.
type CheckinRepository struct {
	db *gorm.DB
}

// NewCheckinRepository creates a new CheckinRepository.
func NewCheckinRepository(db *gorm.DB) *CheckinRepository {
	return &CheckinRepository{db: db}
}

// FindByScheduleID returns all checkins for a given schedule.
func (r *CheckinRepository) FindByScheduleID(scheduleID uint) ([]model.Checkin, error) {
	var checkins []model.Checkin
	if err := r.db.Where("schedule_id = ?", scheduleID).
		Order("checkin_time ASC").
		Find(&checkins).Error; err != nil {
		return nil, err
	}
	return checkins, nil
}

// FindByScheduleAndType returns a checkin for a given schedule and type.
func (r *CheckinRepository) FindByScheduleAndType(scheduleID uint, checkinType string) (*model.Checkin, error) {
	var checkin model.Checkin
	if err := r.db.Where("schedule_id = ? AND type = ?", scheduleID, checkinType).
		First(&checkin).Error; err != nil {
		return nil, err
	}
	return &checkin, nil
}

// Create inserts a new checkin record.
func (r *CheckinRepository) Create(checkin *model.Checkin) error {
	return r.db.Create(checkin).Error
}

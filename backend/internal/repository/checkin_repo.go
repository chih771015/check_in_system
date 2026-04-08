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

// ListAllParams holds optional filter parameters.
type ListAllParams struct {
	DateFrom     string
	DateTo       string
	TranslatorID uint
	CheckinType  string
	IsMakeup     *bool
}

// ListAll returns all checkins with optional filters, joining with users for translator name.
func (r *CheckinRepository) ListAll(params ListAllParams) ([]model.Checkin, error) {
	var checkins []model.Checkin
	q := r.db.Order("checkin_time DESC")
	if params.DateFrom != "" {
		q = q.Where("DATE(checkin_time) >= ?", params.DateFrom)
	}
	if params.DateTo != "" {
		q = q.Where("DATE(checkin_time) <= ?", params.DateTo)
	}
	if params.TranslatorID > 0 {
		q = q.Where("translator_id = ?", params.TranslatorID)
	}
	if params.CheckinType != "" {
		q = q.Where("type = ?", params.CheckinType)
	}
	if params.IsMakeup != nil {
		q = q.Where("is_makeup = ?", *params.IsMakeup)
	}
	if err := q.Find(&checkins).Error; err != nil {
		return nil, err
	}
	return checkins, nil
}

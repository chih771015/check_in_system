package repository

import (
	"context"

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

// WithCtx returns a copy whose *gorm.DB carries the request context so
// the GORM OTel plugin nests SQL spans under the active HTTP span.
func (r *CheckinRepository) WithCtx(ctx context.Context) *CheckinRepository {
	return &CheckinRepository{db: r.db.WithContext(ctx)}
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

// FindByID returns a checkin by ID.
func (r *CheckinRepository) FindByID(id uint) (*model.Checkin, error) {
	var c model.Checkin
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateFields applies a partial update to a checkin record.
func (r *CheckinRepository) UpdateFields(id uint, fields map[string]any) error {
	return r.db.Model(&model.Checkin{}).Where("id = ?", id).Updates(fields).Error
}

// Delete removes a checkin by ID.
func (r *CheckinRepository) Delete(id uint) error {
	return r.db.Delete(&model.Checkin{}, id).Error
}

// DeleteByScheduleID removes all checkins belonging to the given schedule.
// Called before deleting a schedule to satisfy the FK constraint.
func (r *CheckinRepository) DeleteByScheduleID(scheduleID uint) error {
	return r.db.Where("schedule_id = ?", scheduleID).Delete(&model.Checkin{}).Error
}

// DeleteByScheduleIDs removes all checkins belonging to any of the given schedules.
// Called before bulk-deleting a recurrence group.
func (r *CheckinRepository) DeleteByScheduleIDs(scheduleIDs []uint) error {
	if len(scheduleIDs) == 0 {
		return nil
	}
	return r.db.Where("schedule_id IN ?", scheduleIDs).Delete(&model.Checkin{}).Error
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

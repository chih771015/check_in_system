package repository

import (
	"context"

	"translator-checkin/internal/model"

	"gorm.io/gorm"
)

// SchedulePatientRepository handles per-patient slot rows attached to a schedule.
type SchedulePatientRepository struct {
	db *gorm.DB
}

// NewSchedulePatientRepository creates a new SchedulePatientRepository.
func NewSchedulePatientRepository(db *gorm.DB) *SchedulePatientRepository {
	return &SchedulePatientRepository{db: db}
}

// WithCtx returns a copy of the repo bound to ctx so SQL spans nest under the
// caller's HTTP span (see CLAUDE.md).
func (r *SchedulePatientRepository) WithCtx(ctx context.Context) *SchedulePatientRepository {
	return &SchedulePatientRepository{db: r.db.WithContext(ctx)}
}

// CreateBatch inserts multiple SchedulePatients in one statement.
func (r *SchedulePatientRepository) CreateBatch(rows []*model.SchedulePatient) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.Create(&rows).Error
}

// FindByScheduleID returns all SchedulePatients for one schedule with their
// Patient relation preloaded, ordered by start time.
func (r *SchedulePatientRepository) FindByScheduleID(scheduleID uint) ([]model.SchedulePatient, error) {
	var rows []model.SchedulePatient
	err := r.db.
		Preload("Patient").
		Where("schedule_id = ?", scheduleID).
		Order("start_time ASC").
		Find(&rows).Error
	return rows, err
}

// FindByID fetches one SchedulePatient with the Patient preloaded.
func (r *SchedulePatientRepository) FindByID(id uint) (*model.SchedulePatient, error) {
	var sp model.SchedulePatient
	if err := r.db.Preload("Patient").First(&sp, id).Error; err != nil {
		return nil, err
	}
	return &sp, nil
}

// DeleteByScheduleID removes every SchedulePatient row for the given schedule.
func (r *SchedulePatientRepository) DeleteByScheduleID(scheduleID uint) error {
	return r.db.Where("schedule_id = ?", scheduleID).Delete(&model.SchedulePatient{}).Error
}

// DeleteByScheduleIDs removes all SchedulePatients for the given schedule IDs.
// No-op when the slice is empty.
func (r *SchedulePatientRepository) DeleteByScheduleIDs(scheduleIDs []uint) error {
	if len(scheduleIDs) == 0 {
		return nil
	}
	return r.db.Where("schedule_id IN ?", scheduleIDs).Delete(&model.SchedulePatient{}).Error
}

// UpdateStatus sets the status + (optional) no_show_reason of a SchedulePatient.
func (r *SchedulePatientRepository) UpdateStatus(id uint, status, noShowReason string) error {
	return r.db.Model(&model.SchedulePatient{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": status, "no_show_reason": noShowReason}).Error
}

// UpdateActualAmount sets the actual paid amount (整數元) for a SchedulePatient.
func (r *SchedulePatientRepository) UpdateActualAmount(id uint, amount int) error {
	return r.db.Model(&model.SchedulePatient{}).
		Where("id = ?", id).
		Update("actual_amount", amount).Error
}

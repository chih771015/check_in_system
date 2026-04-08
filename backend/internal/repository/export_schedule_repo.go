package repository

import (
	"translator-checkin/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ExportScheduleRepository handles database operations for export schedules.
type ExportScheduleRepository struct {
	db *gorm.DB
}

// NewExportScheduleRepository creates a new ExportScheduleRepository.
func NewExportScheduleRepository(db *gorm.DB) *ExportScheduleRepository {
	return &ExportScheduleRepository{db: db}
}

// Upsert inserts or updates an export schedule for the given admin.
func (r *ExportScheduleRepository) Upsert(es *model.ExportSchedule) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "admin_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"frequency", "day_of_month", "format", "email_to", "enabled", "updated_at"}),
	}).Create(es).Error
}

// FindByAdmin returns the export schedule for a given admin ID.
func (r *ExportScheduleRepository) FindByAdmin(adminID uint) (*model.ExportSchedule, error) {
	var es model.ExportSchedule
	if err := r.db.Where("admin_id = ?", adminID).First(&es).Error; err != nil {
		return nil, err
	}
	return &es, nil
}

// FindAllEnabled returns all enabled export schedules.
func (r *ExportScheduleRepository) FindAllEnabled() ([]model.ExportSchedule, error) {
	var list []model.ExportSchedule
	r.db.Where("enabled = true").Find(&list)
	return list, nil
}

// UpdateLastRun updates the last_run_at timestamp for a given export schedule.
func (r *ExportScheduleRepository) UpdateLastRun(id uint, t interface{}) error {
	return r.db.Model(&model.ExportSchedule{}).Where("id = ?", id).Update("last_run_at", t).Error
}

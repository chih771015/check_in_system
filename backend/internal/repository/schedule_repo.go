package repository

import (
	"translator-checkin/internal/model"

	"gorm.io/gorm"
)

// ScheduleRepository handles database operations for schedules.
type ScheduleRepository struct {
	db *gorm.DB
}

// NewScheduleRepository creates a new ScheduleRepository.
func NewScheduleRepository(db *gorm.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// FindByID returns a schedule by ID with preloaded Translator.
func (r *ScheduleRepository) FindByID(id uint) (*model.Schedule, error) {
	var schedule model.Schedule
	if err := r.db.Preload("Translator").First(&schedule, id).Error; err != nil {
		return nil, err
	}
	return &schedule, nil
}

// FindByTranslator returns schedules for a specific translator within a date range.
func (r *ScheduleRepository) FindByTranslator(translatorID uint, dateFrom, dateTo string) ([]model.Schedule, error) {
	var schedules []model.Schedule
	query := r.db.Preload("Translator").Where("translator_id = ?", translatorID)

	if dateFrom != "" {
		query = query.Where("date >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("date <= ?", dateTo)
	}

	if err := query.Order("date ASC, start_time ASC").Find(&schedules).Error; err != nil {
		return nil, err
	}
	return schedules, nil
}

// FindAll returns schedules with optional filters and preloaded Translator.
func (r *ScheduleRepository) FindAll(translatorID uint, dateFrom, dateTo, location string) ([]model.Schedule, error) {
	var schedules []model.Schedule
	query := r.db.Preload("Translator")

	if translatorID > 0 {
		query = query.Where("translator_id = ?", translatorID)
	}
	if dateFrom != "" {
		query = query.Where("date >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("date <= ?", dateTo)
	}
	if location != "" {
		query = query.Where("location ILIKE ?", "%"+location+"%")
	}

	if err := query.Order("date ASC, start_time ASC").Find(&schedules).Error; err != nil {
		return nil, err
	}
	return schedules, nil
}

// Create inserts a new schedule record.
func (r *ScheduleRepository) Create(schedule *model.Schedule) error {
	return r.db.Create(schedule).Error
}

// CreateBatch inserts multiple schedule records.
func (r *ScheduleRepository) CreateBatch(schedules []*model.Schedule) error {
	return r.db.Create(&schedules).Error
}

// Update saves changes to an existing schedule record.
func (r *ScheduleRepository) Update(schedule *model.Schedule) error {
	return r.db.Save(schedule).Error
}

// Delete removes a schedule by ID.
func (r *ScheduleRepository) Delete(id uint) error {
	return r.db.Delete(&model.Schedule{}, id).Error
}

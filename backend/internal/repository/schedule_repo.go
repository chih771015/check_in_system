package repository

import (
	"context"

	"translator-checkin/internal/model"

	"gorm.io/gorm"
)

// defaultRecentScheduleLimit caps the default "latest created" page when the
// caller does not specify a page size. Mirrors the service-layer constant.
const defaultRecentScheduleLimit = 100

// ScheduleRepository handles database operations for schedules.
type ScheduleRepository struct {
	db *gorm.DB
}

// NewScheduleRepository creates a new ScheduleRepository.
func NewScheduleRepository(db *gorm.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// DB exposes the underlying *gorm.DB so callers (e.g. tests, services that
// need to start a transaction across repositories) can reuse the same handle.
func (r *ScheduleRepository) DB() *gorm.DB {
	return r.db
}

// WithCtx returns a copy whose *gorm.DB carries the request context so
// the GORM OTel plugin nests SQL spans under the active HTTP span.
func (r *ScheduleRepository) WithCtx(ctx context.Context) *ScheduleRepository {
	return &ScheduleRepository{db: r.db.WithContext(ctx)}
}

// FindByID returns a schedule by ID with Translator + Patients (incl. Patient identity) preloaded.
func (r *ScheduleRepository) FindByID(id uint) (*model.Schedule, error) {
	var schedule model.Schedule
	if err := r.db.
		Preload("Translator").
		Preload("Patients.Patient").
		First(&schedule, id).Error; err != nil {
		return nil, err
	}
	return &schedule, nil
}

// FindByTranslator returns schedules for a specific translator within a date range.
func (r *ScheduleRepository) FindByTranslator(translatorID uint, dateFrom, dateTo string) ([]model.Schedule, error) {
	var schedules []model.Schedule
	query := r.db.
		Preload("Translator").
		Preload("Patients.Patient").
		Where("translator_id = ?", translatorID)

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

// FindAll returns schedules with optional filters and Translator + Patients preloaded.
// FindAll returns schedules matching the filters (date ASC) plus the total
// matching count. PageSize <= 0 means "no limit" — return every matching row
// (used by the nightly reminder job). Otherwise the result is one page.
func (r *ScheduleRepository) FindAll(translatorID uint, dateFrom, dateTo, location string, page, pageSize int) ([]model.Schedule, int64, error) {
	var schedules []model.Schedule
	var total int64

	base := r.db.Model(&model.Schedule{})
	if translatorID > 0 {
		base = base.Where("translator_id = ?", translatorID)
	}
	if dateFrom != "" {
		base = base.Where("date >= ?", dateFrom)
	}
	if dateTo != "" {
		base = base.Where("date <= ?", dateTo)
	}
	if location != "" {
		base = base.Where("location ILIKE ?", "%"+location+"%")
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := base.
		Preload("Translator").
		Preload("Patients.Patient").
		Order("date ASC, start_time ASC")
	if pageSize > 0 {
		if page <= 0 {
			page = 1
		}
		query = query.Limit(pageSize).Offset((page - 1) * pageSize)
	}
	if err := query.Find(&schedules).Error; err != nil {
		return nil, 0, err
	}
	return schedules, total, nil
}

// FindRecentByCreated returns one page of schedules ordered by created_at DESC
// (id DESC as a stable tiebreaker), with Translator + Patients preloaded, plus
// the total row count. This backs the default (unfiltered) admin list view:
// "latest created schedules".
func (r *ScheduleRepository) FindRecentByCreated(page, pageSize int) ([]model.Schedule, int64, error) {
	var schedules []model.Schedule
	var total int64
	if err := r.db.Model(&model.Schedule{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if pageSize <= 0 {
		pageSize = defaultRecentScheduleLimit
	}
	if page <= 0 {
		page = 1
	}
	if err := r.db.
		Preload("Translator").
		Preload("Patients.Patient").
		Order("created_at DESC, id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&schedules).Error; err != nil {
		return nil, 0, err
	}
	return schedules, total, nil
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

// IDsByRecurrenceGroup returns the IDs of every schedule sharing the given
// recurrence group, so the caller can delete related checkins first.
func (r *ScheduleRepository) IDsByRecurrenceGroup(groupID string) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.Schedule{}).
		Where("recurrence_group_id = ?", groupID).
		Pluck("id", &ids).Error
	return ids, err
}

// DeleteByRecurrenceGroup deletes every schedule sharing the given recurrence
// group id and returns the number of rows removed.
func (r *ScheduleRepository) DeleteByRecurrenceGroup(groupID string) (int64, error) {
	res := r.db.Where("recurrence_group_id = ?", groupID).Delete(&model.Schedule{})
	return res.RowsAffected, res.Error
}

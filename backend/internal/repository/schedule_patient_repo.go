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

// SumActualByPatients returns, per patient id, the all-time sum of
// actual_amount across their schedule_patient rows. Patients with no rows are
// absent from the map (callers treat missing as 0). Used by the patient Excel
// export to append a "實付金額總額" column in one batched query (no N+1).
func (r *SchedulePatientRepository) SumActualByPatients(patientIDs []uint) (map[uint]int64, error) {
	out := map[uint]int64{}
	if len(patientIDs) == 0 {
		return out, nil
	}
	type row struct {
		PatientID uint
		Total     int64
	}
	var rows []row
	err := r.db.Model(&model.SchedulePatient{}).
		Select("patient_id, COALESCE(SUM(actual_amount), 0) AS total").
		Where("patient_id IN ?", patientIDs).
		Group("patient_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.PatientID] = x.Total
	}
	return out, nil
}

// sumActualInRange sums actual_amount over schedules whose date falls in the
// half-open range [fromInclusive, toExclusive) (both YYYY-MM-DD). Half-open
// avoids the upper-bound edge issue of an inclusive string compare against a
// datetime-formatted date column. When patientID is 0 it sums across all
// patients; otherwise it scopes to that patient. Single query body shared by
// the two public wrappers below.
func (r *SchedulePatientRepository) sumActualInRange(patientID uint, fromInclusive, toExclusive string) (int64, error) {
	q := r.db.Table("schedule_patients as sp").
		Joins("JOIN schedules ON schedules.id = sp.schedule_id").
		Where("schedules.date >= ? AND schedules.date < ?", fromInclusive, toExclusive)
	if patientID != 0 {
		q = q.Where("sp.patient_id = ?", patientID)
	}
	var total int64
	err := q.Select("COALESCE(SUM(sp.actual_amount), 0)").Scan(&total).Error
	return total, err
}

// SumActualByDateRange returns the actual_amount total across all patients for
// the half-open range [fromInclusive, toExclusive). Backs the admin "current
// month total expenditure" banner.
func (r *SchedulePatientRepository) SumActualByDateRange(fromInclusive, toExclusive string) (int64, error) {
	return r.sumActualInRange(0, fromInclusive, toExclusive)
}

// SumActualByPatientDateRange returns the sum of actual_amount for one patient
// over the half-open range [fromInclusive, toExclusive). Backs the "patient's
// actual-paid total for a year" hint shown at scheduling.
func (r *SchedulePatientRepository) SumActualByPatientDateRange(patientID uint, fromInclusive, toExclusive string) (int64, error) {
	return r.sumActualInRange(patientID, fromInclusive, toExclusive)
}

// UpdateActualAmount sets the actual paid amount (整數元) for a SchedulePatient.
func (r *SchedulePatientRepository) UpdateActualAmount(id uint, amount int) error {
	return r.db.Model(&model.SchedulePatient{}).
		Where("id = ?", id).
		Update("actual_amount", amount).Error
}

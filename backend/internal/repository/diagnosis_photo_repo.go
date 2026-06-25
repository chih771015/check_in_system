package repository

import (
	"context"

	"translator-checkin/internal/model"

	"gorm.io/gorm"
)

// DiagnosisPhotoRepository manages DiagnosisPhoto rows linked to a
// SchedulePatient slot. Each slot allows up to 3 photos — enforcement of that
// cap lives in the service layer.
type DiagnosisPhotoRepository struct {
	db *gorm.DB
}

// NewDiagnosisPhotoRepository creates a new DiagnosisPhotoRepository.
func NewDiagnosisPhotoRepository(db *gorm.DB) *DiagnosisPhotoRepository {
	return &DiagnosisPhotoRepository{db: db}
}

// WithCtx returns a copy bound to ctx so OTel SQL spans nest under the caller.
func (r *DiagnosisPhotoRepository) WithCtx(ctx context.Context) *DiagnosisPhotoRepository {
	return &DiagnosisPhotoRepository{db: r.db.WithContext(ctx)}
}

// Create inserts a new diagnosis photo row.
func (r *DiagnosisPhotoRepository) Create(p *model.DiagnosisPhoto) error {
	return r.db.Create(p).Error
}

// FindBySchedulePatientID returns all photos for one (schedule, patient) slot,
// ordered by upload time ascending.
func (r *DiagnosisPhotoRepository) FindBySchedulePatientID(spID uint) ([]model.DiagnosisPhoto, error) {
	var photos []model.DiagnosisPhoto
	err := r.db.Where("schedule_patient_id = ?", spID).
		Order("uploaded_at ASC").
		Find(&photos).Error
	return photos, err
}

// FindBySchedulePatientIDs returns all photos for the given slot ids in one
// query (ordered by upload time), letting callers avoid an N+1 when building a
// patient's history. An empty id list short-circuits to an empty slice.
func (r *DiagnosisPhotoRepository) FindBySchedulePatientIDs(ids []uint) ([]model.DiagnosisPhoto, error) {
	var photos []model.DiagnosisPhoto
	if len(ids) == 0 {
		return photos, nil
	}
	err := r.db.Where("schedule_patient_id IN ?", ids).
		Order("uploaded_at ASC").
		Find(&photos).Error
	return photos, err
}

// FindByID returns a single diagnosis photo by its primary key.
func (r *DiagnosisPhotoRepository) FindByID(id uint) (*model.DiagnosisPhoto, error) {
	var photo model.DiagnosisPhoto
	if err := r.db.First(&photo, id).Error; err != nil {
		return nil, err
	}
	return &photo, nil
}

// Delete removes a diagnosis photo row by its primary key.
func (r *DiagnosisPhotoRepository) Delete(id uint) error {
	return r.db.Delete(&model.DiagnosisPhoto{}, id).Error
}

// CountBySchedulePatientID returns how many photos already exist for one slot.
// Used by the service to enforce the 3-photo cap before insert.
func (r *DiagnosisPhotoRepository) CountBySchedulePatientID(spID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.DiagnosisPhoto{}).
		Where("schedule_patient_id = ?", spID).
		Count(&n).Error
	return n, err
}

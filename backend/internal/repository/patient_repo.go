package repository

import (
	"context"

	"translator-checkin/internal/model"

	"gorm.io/gorm"
)

// PatientRepository handles database operations for patients.
type PatientRepository struct {
	db *gorm.DB
}

// NewPatientRepository creates a new PatientRepository.
func NewPatientRepository(db *gorm.DB) *PatientRepository {
	return &PatientRepository{db: db}
}

// WithCtx returns a copy whose *gorm.DB carries the request context so the
// GORM OTel plugin nests SQL spans under the active HTTP span.
func (r *PatientRepository) WithCtx(ctx context.Context) *PatientRepository {
	return &PatientRepository{db: r.db.WithContext(ctx)}
}

// Create inserts a new patient record. Callers must uppercase IDNumber before
// invoking this method (see service.PatientService).
func (r *PatientRepository) Create(patient *model.Patient) error {
	return r.db.Create(patient).Error
}

// Update saves changes to an existing patient record.
func (r *PatientRepository) Update(patient *model.Patient) error {
	return r.db.Save(patient).Error
}

// Delete hard-deletes a patient by primary key.
func (r *PatientRepository) Delete(id uint) error {
	return r.db.Delete(&model.Patient{}, id).Error
}

// FindByID returns a patient by ID.
func (r *PatientRepository) FindByID(id uint) (*model.Patient, error) {
	var patient model.Patient
	if err := r.db.First(&patient, id).Error; err != nil {
		return nil, err
	}
	return &patient, nil
}

// FindByIDTypeAndNumber returns the patient matching the given identity tuple
// or gorm.ErrRecordNotFound if none. Used by the service to enforce the
// (id_type, id_number) uniqueness rule before insert/update.
func (r *PatientRepository) FindByIDTypeAndNumber(idType, idNumber string) (*model.Patient, error) {
	var patient model.Patient
	if err := r.db.Where("id_type = ? AND id_number = ?", idType, idNumber).
		First(&patient).Error; err != nil {
		return nil, err
	}
	return &patient, nil
}

// List returns a page of patients filtered by `search` (matches name / phone /
// id_number, case-insensitive) plus the total row count for pagination. If
// pageSize is <= 0 it defaults to 20; page defaults to 1.
func (r *PatientRepository) List(search string, page, pageSize int) ([]model.Patient, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	query := r.db.Model(&model.Patient{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name ILIKE ? OR phone ILIKE ? OR id_number ILIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var patients []model.Patient
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&patients).Error; err != nil {
		return nil, 0, err
	}
	return patients, total, nil
}

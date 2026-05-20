package service

import (
	"context"
	"errors"
	"strings"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"gorm.io/gorm"
)

// ErrPatientDuplicate is returned when a Create/Update would collide with an
// existing patient on (id_type, id_number). Handlers map this to 409 / a
// dedicated error code.
var ErrPatientDuplicate = errors.New("patient with same id_type and id_number already exists")

// ErrPatientNotFound is returned when a patient lookup misses.
var ErrPatientNotFound = errors.New("patient not found")

// PatientService implements CRUD for patients plus the visit-history
// aggregation hook (stub in stage 2, real in stage 4).
type PatientService struct {
	patientRepo *repository.PatientRepository
	// Stage 3: when set, ListForTranslator restricts results to patients
	// the caller actually has in their schedules.
	spRepo *repository.SchedulePatientRepository
	// Stage 4: history aggregation deps.
	scheduleRepo *repository.ScheduleRepository
	photoRepo    *repository.DiagnosisPhotoRepository
}

// NewPatientService creates a new PatientService.
func NewPatientService(patientRepo *repository.PatientRepository) *PatientService {
	return &PatientService{patientRepo: patientRepo}
}

// WithScopeRepo wires up SchedulePatientRepository so ListForTranslator can
// restrict results to the caller's own schedules. Returns the service for
// chaining.
func (s *PatientService) WithScopeRepo(spRepo *repository.SchedulePatientRepository) *PatientService {
	s.spRepo = spRepo
	return s
}

// WithHistoryRepos wires up Schedule + SchedulePatient + DiagnosisPhoto repos
// so GetHistory can return real visit data.
func (s *PatientService) WithHistoryRepos(
	scheduleRepo *repository.ScheduleRepository,
	spRepo *repository.SchedulePatientRepository,
	photoRepo *repository.DiagnosisPhotoRepository,
) *PatientService {
	s.scheduleRepo = scheduleRepo
	s.spRepo = spRepo
	s.photoRepo = photoRepo
	return s
}

// normalizeIDNumber uppercases and trims the ID number so that lookups are
// case- and whitespace-insensitive without changing user-supplied display.
func normalizeIDNumber(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// Create inserts a new patient after checking for duplicates on
// (id_type, id_number). Returns ErrPatientDuplicate if a collision is found.
func (s *PatientService) Create(ctx context.Context, req dto.CreatePatientRequest) (*model.Patient, error) {
	repo := s.patientRepo.WithCtx(ctx)
	idNumber := normalizeIDNumber(req.IDNumber)

	if existing, err := repo.FindByIDTypeAndNumber(req.IDType, idNumber); err == nil && existing != nil {
		return nil, ErrPatientDuplicate
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	patient := &model.Patient{
		Name:     strings.TrimSpace(req.Name),
		Phone:    strings.TrimSpace(req.Phone),
		IDType:   req.IDType,
		IDNumber: idNumber,
	}
	if err := repo.Create(patient); err != nil {
		return nil, err
	}
	return patient, nil
}

// Update edits an existing patient. The duplicate check ignores the current
// record so a no-op update still works.
func (s *PatientService) Update(ctx context.Context, id uint, req dto.UpdatePatientRequest) (*model.Patient, error) {
	repo := s.patientRepo.WithCtx(ctx)
	patient, err := repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPatientNotFound
		}
		return nil, err
	}

	idNumber := normalizeIDNumber(req.IDNumber)
	if existing, err := repo.FindByIDTypeAndNumber(req.IDType, idNumber); err == nil && existing != nil && existing.ID != id {
		return nil, ErrPatientDuplicate
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	patient.Name = strings.TrimSpace(req.Name)
	patient.Phone = strings.TrimSpace(req.Phone)
	patient.IDType = req.IDType
	patient.IDNumber = idNumber

	if err := repo.Update(patient); err != nil {
		return nil, err
	}
	return patient, nil
}

// Delete removes a patient by ID.
func (s *PatientService) Delete(ctx context.Context, id uint) error {
	repo := s.patientRepo.WithCtx(ctx)
	if _, err := repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPatientNotFound
		}
		return err
	}
	return repo.Delete(id)
}

// FindByID exposes the underlying repository lookup so handlers don't have to
// touch the repo directly.
func (s *PatientService) FindByID(ctx context.Context, id uint) (*model.Patient, error) {
	repo := s.patientRepo.WithCtx(ctx)
	patient, err := repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPatientNotFound
		}
		return nil, err
	}
	return patient, nil
}

// List returns a page of patients with total count.
func (s *PatientService) List(ctx context.Context, q dto.PatientListQuery) ([]model.Patient, int64, error) {
	return s.patientRepo.WithCtx(ctx).List(strings.TrimSpace(q.Search), q.Page, q.PageSize)
}

// ListForTranslator returns the patient list a translator may see — only
// patients that appear in their own schedules (via schedule_patients).
//
// When WithScopeRepo has not been called the service falls back to listing
// all patients (legacy stage-2 behaviour for tests that don't wire scope).
func (s *PatientService) ListForTranslator(ctx context.Context, translatorID uint, q dto.PatientListQuery) ([]model.Patient, int64, error) {
	if s.spRepo == nil {
		return s.patientRepo.WithCtx(ctx).List(strings.TrimSpace(q.Search), q.Page, q.PageSize)
	}
	return s.patientRepo.WithCtx(ctx).ListForTranslator(translatorID, strings.TrimSpace(q.Search), q.Page, q.PageSize)
}

// GetHistory returns the visit history for a single patient.
// GetHistory returns a patient's visit history aggregated from
// schedule_patients + schedules + diagnosis_photos, ordered by date DESC.
//
// Stage 4 implements the real aggregation; if history repos have not been
// wired (legacy stage-2 caller) it falls back to an empty slice.
func (s *PatientService) GetHistory(ctx context.Context, patientID uint) (*dto.PatientHistoryResponse, error) {
	patient, err := s.FindByID(ctx, patientID)
	if err != nil {
		return nil, err
	}

	entries := []dto.PatientHistoryEntry{}
	if s.scheduleRepo != nil && s.spRepo != nil && s.photoRepo != nil {
		entries, err = s.buildHistoryEntries(ctx, patientID)
		if err != nil {
			return nil, err
		}
	}

	return &dto.PatientHistoryResponse{
		Patient: dto.PatientResponse{
			ID:        patient.ID,
			Name:      patient.Name,
			Phone:     patient.Phone,
			IDType:    patient.IDType,
			IDNumber:  patient.IDNumber,
			CreatedAt: patient.CreatedAt,
			UpdatedAt: patient.UpdatedAt,
		},
		History: entries,
	}, nil
}

// buildHistoryEntries does the real DB walk for GetHistory.
func (s *PatientService) buildHistoryEntries(ctx context.Context, patientID uint) ([]dto.PatientHistoryEntry, error) {
	db := s.scheduleRepo.DB().WithContext(ctx)
	// Pull every schedule_patient row for this patient, sorted by schedule date desc.
	type joined struct {
		SPID         uint
		ScheduleID   uint
		Date         string
		SPStart      string
		SPEnd        string
		Location     string
		Status       string
		NoShowReason string
		TName        string
	}
	var rows []joined
	err := db.Table("schedule_patients as sp").
		Select(`sp.id as sp_id, sp.schedule_id, sp.start_time as sp_start, sp.end_time as sp_end,
			sp.status, sp.no_show_reason,
			schedules.date, schedules.location, users.name as t_name`).
		Joins("JOIN schedules ON schedules.id = sp.schedule_id").
		Joins("JOIN users ON users.id = schedules.translator_id").
		Where("sp.patient_id = ?", patientID).
		Order("schedules.date DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	entries := make([]dto.PatientHistoryEntry, 0, len(rows))
	for _, r := range rows {
		photos, _ := s.photoRepo.WithCtx(ctx).FindBySchedulePatientID(r.SPID)
		photoURLs := make([]string, 0, len(photos))
		for _, p := range photos {
			photoURLs = append(photoURLs, p.PhotoURL)
		}
		// sqlite returns date as RFC3339; postgres returns YYYY-MM-DD. Trim T... if present.
		dateOnly := r.Date
		if idx := indexT(dateOnly); idx > 0 {
			dateOnly = dateOnly[:idx]
		}
		entries = append(entries, dto.PatientHistoryEntry{
			ScheduleID:      r.ScheduleID,
			Date:            dateOnly,
			StartTime:       r.SPStart,
			EndTime:         r.SPEnd,
			Location:        r.Location,
			TranslatorName:  r.TName,
			Status:          r.Status,
			NoShowReason:    r.NoShowReason,
			DiagnosisPhotos: photoURLs,
		})
	}
	return entries, nil
}

// indexT returns the index of 'T' in s or -1 if absent.
func indexT(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == 'T' {
			return i
		}
	}
	return -1
}

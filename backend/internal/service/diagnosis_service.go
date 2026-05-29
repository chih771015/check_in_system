package service

import (
	"context"
	"errors"
	"time"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"
)

// Sentinel errors returned by DiagnosisService.
var (
	ErrSchedulePatientNotFound = errors.New("schedule patient not found")
	ErrDiagnosisPhotoLimit     = errors.New("diagnosis photo limit reached (max 3 per patient)")
	ErrDiagnosisNotOwned       = errors.New("schedule patient does not belong to this translator")
	ErrNoShowReasonRequired    = errors.New("no_show_reason is required")
)

// MaxDiagnosisPhotos is the per-(schedule, patient) cap defined in the spec.
const MaxDiagnosisPhotos = 3

// DiagnosisService handles per-patient diagnosis photo uploads and "no show"
// status updates inside a schedule. Both flows are translator-only by default
// (admin surrogate variants are exposed via Admin*).
type DiagnosisService struct {
	spRepo       *repository.SchedulePatientRepository
	photoRepo    *repository.DiagnosisPhotoRepository
	scheduleRepo *repository.ScheduleRepository
}

// NewDiagnosisService creates a new DiagnosisService.
func NewDiagnosisService(
	spRepo *repository.SchedulePatientRepository,
	photoRepo *repository.DiagnosisPhotoRepository,
	scheduleRepo *repository.ScheduleRepository,
) *DiagnosisService {
	return &DiagnosisService{spRepo: spRepo, photoRepo: photoRepo, scheduleRepo: scheduleRepo}
}

// UploadDiagnosis appends new photo URLs to a SchedulePatient and marks it
// completed when at least one photo is present.
//   - translatorID must own the parent schedule
//   - existing photos + new photos must not exceed MaxDiagnosisPhotos
func (s *DiagnosisService) UploadDiagnosis(ctx context.Context, translatorID, spID uint, photoURLs []string) error {
	sp, err := s.assertOwnedSchedulePatient(ctx, translatorID, spID)
	if err != nil {
		return err
	}

	photoRepo := s.photoRepo.WithCtx(ctx)
	existing, err := photoRepo.CountBySchedulePatientID(sp.ID)
	if err != nil {
		return err
	}
	if int(existing)+len(photoURLs) > MaxDiagnosisPhotos {
		return ErrDiagnosisPhotoLimit
	}

	now := time.Now()
	for _, url := range photoURLs {
		if err := photoRepo.Create(&model.DiagnosisPhoto{
			SchedulePatientID: sp.ID,
			PhotoURL:          url,
			UploadedAt:        now,
		}); err != nil {
			return err
		}
	}

	// Mark slot completed (overrides any prior status — admin re-upload is allowed).
	return s.spRepo.WithCtx(ctx).UpdateStatus(sp.ID, model.SchedulePatientStatusCompleted, "")
}

// MarkNoShow records that a patient did not show up at their slot.
// reason is required to keep ops accountable.
func (s *DiagnosisService) MarkNoShow(ctx context.Context, translatorID, spID uint, reason string) error {
	if reason == "" {
		return ErrNoShowReasonRequired
	}
	sp, err := s.assertOwnedSchedulePatient(ctx, translatorID, spID)
	if err != nil {
		return err
	}
	return s.spRepo.WithCtx(ctx).UpdateStatus(sp.ID, model.SchedulePatientStatusNoShow, reason)
}

// AdminUploadDiagnosis is the admin-surrogate variant — no ownership check.
func (s *DiagnosisService) AdminUploadDiagnosis(ctx context.Context, spID uint, photoURLs []string) error {
	sp, err := s.spRepo.WithCtx(ctx).FindByID(spID)
	if err != nil {
		return ErrSchedulePatientNotFound
	}
	photoRepo := s.photoRepo.WithCtx(ctx)
	existing, err := photoRepo.CountBySchedulePatientID(sp.ID)
	if err != nil {
		return err
	}
	if int(existing)+len(photoURLs) > MaxDiagnosisPhotos {
		return ErrDiagnosisPhotoLimit
	}
	now := time.Now()
	for _, url := range photoURLs {
		if err := photoRepo.Create(&model.DiagnosisPhoto{
			SchedulePatientID: sp.ID,
			PhotoURL:          url,
			UploadedAt:        now,
		}); err != nil {
			return err
		}
	}
	return s.spRepo.WithCtx(ctx).UpdateStatus(sp.ID, model.SchedulePatientStatusCompleted, "")
}

// AdminMarkNoShow is the admin-surrogate variant — no ownership check.
func (s *DiagnosisService) AdminMarkNoShow(ctx context.Context, spID uint, reason string) error {
	if reason == "" {
		return ErrNoShowReasonRequired
	}
	if _, err := s.spRepo.WithCtx(ctx).FindByID(spID); err != nil {
		return ErrSchedulePatientNotFound
	}
	return s.spRepo.WithCtx(ctx).UpdateStatus(spID, model.SchedulePatientStatusNoShow, reason)
}

// assertOwnedSchedulePatient loads a SchedulePatient and verifies that its
// parent schedule is owned by the given translator.
func (s *DiagnosisService) assertOwnedSchedulePatient(ctx context.Context, translatorID, spID uint) (*model.SchedulePatient, error) {
	sp, err := s.spRepo.WithCtx(ctx).FindByID(spID)
	if err != nil {
		return nil, ErrSchedulePatientNotFound
	}
	schedule, err := s.scheduleRepo.WithCtx(ctx).FindByID(sp.ScheduleID)
	if err != nil {
		return nil, ErrScheduleNotFound
	}
	if schedule.TranslatorID != translatorID {
		return nil, ErrDiagnosisNotOwned
	}
	return sp, nil
}

// GetPhotos returns the diagnosis photo URLs attached to a SchedulePatient,
// ordered by upload time. Used by the admin schedule detail modal to surface
// each completed patient's certificate files.
func (s *DiagnosisService) GetPhotos(ctx context.Context, spID uint) ([]string, error) {
	if _, err := s.spRepo.WithCtx(ctx).FindByID(spID); err != nil {
		return nil, ErrSchedulePatientNotFound
	}
	photos, err := s.photoRepo.WithCtx(ctx).FindBySchedulePatientID(spID)
	if err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(photos))
	for _, p := range photos {
		urls = append(urls, p.PhotoURL)
	}
	return urls, nil
}

// ListResults returns the paginated overview of all "terminal" SchedulePatient
// rows (status = completed or no_show) — admins use this to see every visit
// outcome sorted by schedule date and patient slot, most recent first.
//
// We deliberately exclude `pending` rows: they aren't a result yet.
func (s *DiagnosisService) ListResults(ctx context.Context, q dto.DiagnosisResultsQuery) (*dto.DiagnosisResultsResponse, error) {
	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	db := s.scheduleRepo.DB().WithContext(ctx)

	// Build the base query once and reuse for count + data.
	base := db.Table("schedule_patients AS sp").
		Joins("JOIN schedules s ON s.id = sp.schedule_id").
		Joins("JOIN users u ON u.id = s.translator_id").
		Joins("JOIN patients p ON p.id = sp.patient_id").
		Where("sp.status IN ?", []string{
			model.SchedulePatientStatusCompleted,
			model.SchedulePatientStatusNoShow,
		})

	if q.Status == model.SchedulePatientStatusCompleted ||
		q.Status == model.SchedulePatientStatusNoShow {
		base = base.Where("sp.status = ?", q.Status)
	}
	if q.TranslatorID > 0 {
		base = base.Where("s.translator_id = ?", q.TranslatorID)
	}
	if q.DateFrom != "" {
		base = base.Where("s.date >= ?", q.DateFrom)
	}
	if q.DateTo != "" {
		base = base.Where("s.date <= ?", q.DateTo)
	}
	if q.PatientName != "" {
		base = base.Where("p.name LIKE ?", "%"+q.PatientName+"%")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}

	type row struct {
		SPID           uint      `gorm:"column:sp_id"`
		ScheduleID     uint      `gorm:"column:schedule_id"`
		Date           string    `gorm:"column:date"`
		SPStart        string    `gorm:"column:sp_start"`
		SPEnd          string    `gorm:"column:sp_end"`
		Location       string    `gorm:"column:location"`
		Note           string    `gorm:"column:note"`
		Status         string    `gorm:"column:status"`
		NoShowReason   string    `gorm:"column:no_show_reason"`
		UpdatedAt      time.Time `gorm:"column:updated_at"`
		TranslatorID   uint      `gorm:"column:translator_id"`
		TranslatorName string    `gorm:"column:translator_name"`
		PatientID      uint      `gorm:"column:patient_id"`
		PatientName    string    `gorm:"column:patient_name"`
		PatientPhone   string    `gorm:"column:patient_phone"`
		IDType         string    `gorm:"column:id_type"`
		IDNumber       string    `gorm:"column:id_number"`
	}
	var rows []row
	err := base.
		Select(`sp.id AS sp_id, sp.schedule_id, sp.start_time AS sp_start, sp.end_time AS sp_end,
			sp.status, sp.no_show_reason, sp.updated_at,
			s.date, s.location, s.note,
			s.translator_id, u.name AS translator_name,
			sp.patient_id, p.name AS patient_name, p.phone AS patient_phone,
			p.id_type, p.id_number`).
		Order("s.date DESC, sp.start_time DESC, sp.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// Batch-load diagnosis photos for all rows on this page to avoid N+1.
	spIDs := make([]uint, 0, len(rows))
	for _, r := range rows {
		spIDs = append(spIDs, r.SPID)
	}
	photosByID := map[uint][]string{}
	if len(spIDs) > 0 {
		var allPhotos []model.DiagnosisPhoto
		if err := db.Where("schedule_patient_id IN ?", spIDs).
			Order("uploaded_at ASC").
			Find(&allPhotos).Error; err != nil {
			return nil, err
		}
		for _, p := range allPhotos {
			photosByID[p.SchedulePatientID] = append(photosByID[p.SchedulePatientID], p.PhotoURL)
		}
	}

	entries := make([]dto.DiagnosisResultEntry, 0, len(rows))
	for _, r := range rows {
		dateOnly := r.Date
		if i := indexOfByte(dateOnly, 'T'); i > 0 {
			dateOnly = dateOnly[:i]
		}
		entries = append(entries, dto.DiagnosisResultEntry{
			SchedulePatientID: r.SPID,
			ScheduleID:        r.ScheduleID,
			Date:              dateOnly,
			StartTime:         r.SPStart,
			EndTime:           r.SPEnd,
			Location:          r.Location,
			Note:              r.Note,
			TranslatorID:      r.TranslatorID,
			TranslatorName:    r.TranslatorName,
			PatientID:         r.PatientID,
			PatientName:       r.PatientName,
			PatientPhone:      r.PatientPhone,
			IDType:            r.IDType,
			IDNumber:          r.IDNumber,
			Status:            r.Status,
			NoShowReason:      r.NoShowReason,
			DiagnosisPhotos:   photosByID[r.SPID],
			UpdatedAt:         r.UpdatedAt,
		})
	}

	return &dto.DiagnosisResultsResponse{
		Data:     entries,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func indexOfByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

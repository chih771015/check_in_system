package service

import (
	"context"
	"errors"
	"time"

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

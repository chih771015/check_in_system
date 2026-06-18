package service

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"translator-checkin/internal/config"
	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/xuri/excelize/v2"
)

// Sentinel errors returned by DiagnosisService.
var (
	ErrSchedulePatientNotFound   = errors.New("schedule patient not found")
	ErrDiagnosisPhotoLimit       = errors.New("diagnosis photo limit reached (max 30 per patient)")
	ErrDiagnosisNotOwned         = errors.New("schedule patient does not belong to this translator")
	ErrDiagnosisPhotoNotFound    = errors.New("diagnosis photo not found")
	ErrDiagnosisLockedAfterLeave = errors.New("diagnosis can no longer be changed after leave check-in")
	ErrNoShowReasonRequired      = errors.New("no_show_reason is required")
)

// MaxDiagnosisPhotos is the per-(schedule, patient) cap defined in the spec.
const MaxDiagnosisPhotos = 30

// DiagnosisService handles per-patient diagnosis photo uploads and "no show"
// status updates inside a schedule. Both flows are translator-only by default
// (admin surrogate variants are exposed via Admin*).
type DiagnosisService struct {
	spRepo       *repository.SchedulePatientRepository
	photoRepo    *repository.DiagnosisPhotoRepository
	scheduleRepo *repository.ScheduleRepository
	checkinRepo  *repository.CheckinRepository // optional; enables the post-leave lock
}

// NewDiagnosisService creates a new DiagnosisService.
func NewDiagnosisService(
	spRepo *repository.SchedulePatientRepository,
	photoRepo *repository.DiagnosisPhotoRepository,
	scheduleRepo *repository.ScheduleRepository,
) *DiagnosisService {
	return &DiagnosisService{spRepo: spRepo, photoRepo: photoRepo, scheduleRepo: scheduleRepo}
}

// WithCheckinRepo wires the checkin repo so translator-side edits are locked
// once the schedule has a "leave" check-in. When not set, the lock is inactive
// (keeps legacy constructors / tests simple).
func (s *DiagnosisService) WithCheckinRepo(r *repository.CheckinRepository) *DiagnosisService {
	s.checkinRepo = r
	return s
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
	// Note: uploads are intentionally allowed after the leave check-in —
	// late-arriving evidence (X-ray / lab results) can be appended. Only
	// delete / no_show stay locked post-leave (see DeletePhoto / MarkNoShow).

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
	if err := s.assertNotLeftYet(ctx, sp.ScheduleID); err != nil {
		return err
	}
	// Marking no_show means "I pressed completed by mistake" — drop any photos
	// so a no_show slot never carries stale evidence.
	if err := s.purgePhotos(ctx, sp.ID); err != nil {
		return err
	}
	// No-show means nothing was paid → actual amount is 0.
	if err := s.spRepo.WithCtx(ctx).UpdateActualAmount(sp.ID, 0); err != nil {
		return err
	}
	return s.spRepo.WithCtx(ctx).UpdateStatus(sp.ID, model.SchedulePatientStatusNoShow, reason)
}

// SetActualAmount records the actual paid amount (整數元) a translator entered
// after the visit. Ownership is enforced; not subject to the leave lock since
// it is post-visit data entry (like appending late results).
func (s *DiagnosisService) SetActualAmount(ctx context.Context, translatorID, spID uint, amount int) error {
	sp, err := s.assertOwnedSchedulePatient(ctx, translatorID, spID)
	if err != nil {
		return err
	}
	if amount < 0 {
		amount = 0
	}
	return s.spRepo.WithCtx(ctx).UpdateActualAmount(sp.ID, amount)
}

// AdminSetActualAmount is the admin-surrogate variant — no ownership check.
func (s *DiagnosisService) AdminSetActualAmount(ctx context.Context, spID uint, amount int) error {
	if _, err := s.spRepo.WithCtx(ctx).FindByID(spID); err != nil {
		return ErrSchedulePatientNotFound
	}
	if amount < 0 {
		amount = 0
	}
	return s.spRepo.WithCtx(ctx).UpdateActualAmount(spID, amount)
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
	if err := s.purgePhotos(ctx, spID); err != nil {
		return err
	}
	if err := s.spRepo.WithCtx(ctx).UpdateActualAmount(spID, 0); err != nil {
		return err
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

// ListPhotoItems returns the diagnosis photos (with their row IDs) for a
// SchedulePatient owned by the given translator. The IDs let the client delete
// a specific photo. Ownership is enforced.
func (s *DiagnosisService) ListPhotoItems(ctx context.Context, translatorID, spID uint) ([]dto.DiagnosisPhotoItem, error) {
	sp, err := s.assertOwnedSchedulePatient(ctx, translatorID, spID)
	if err != nil {
		return nil, err
	}
	return s.photoItems(ctx, sp.ID)
}

// AdminListPhotoItems is the admin-surrogate variant of ListPhotoItems — no
// ownership check.
func (s *DiagnosisService) AdminListPhotoItems(ctx context.Context, spID uint) ([]dto.DiagnosisPhotoItem, error) {
	if _, err := s.spRepo.WithCtx(ctx).FindByID(spID); err != nil {
		return nil, ErrSchedulePatientNotFound
	}
	return s.photoItems(ctx, spID)
}

func (s *DiagnosisService) photoItems(ctx context.Context, spID uint) ([]dto.DiagnosisPhotoItem, error) {
	photos, err := s.photoRepo.WithCtx(ctx).FindBySchedulePatientID(spID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.DiagnosisPhotoItem, 0, len(photos))
	for _, p := range photos {
		items = append(items, dto.DiagnosisPhotoItem{ID: p.ID, PhotoURL: p.PhotoURL})
	}
	return items, nil
}

// DeletePhoto removes one diagnosis photo owned (transitively) by the given
// translator. After deletion, if the slot has no photos left, its status is
// reverted to "pending" so the translator can re-upload or mark no-show again.
// The underlying file is removed best-effort.
func (s *DiagnosisService) DeletePhoto(ctx context.Context, translatorID, photoID uint) error {
	photo, err := s.photoRepo.WithCtx(ctx).FindByID(photoID)
	if err != nil {
		return ErrDiagnosisPhotoNotFound
	}
	sp, err := s.assertOwnedSchedulePatient(ctx, translatorID, photo.SchedulePatientID)
	if err != nil {
		return err
	}
	if err := s.assertNotLeftYet(ctx, sp.ScheduleID); err != nil {
		return err
	}
	return s.deletePhotoRow(ctx, photo)
}

// AdminDeletePhoto is the admin-surrogate variant of DeletePhoto — no ownership
// check.
func (s *DiagnosisService) AdminDeletePhoto(ctx context.Context, photoID uint) error {
	photo, err := s.photoRepo.WithCtx(ctx).FindByID(photoID)
	if err != nil {
		return ErrDiagnosisPhotoNotFound
	}
	if _, err := s.spRepo.WithCtx(ctx).FindByID(photo.SchedulePatientID); err != nil {
		return ErrSchedulePatientNotFound
	}
	return s.deletePhotoRow(ctx, photo)
}

// assertNotLeftYet rejects translator-side diagnosis edits once the schedule
// has a "leave" check-in: after departure only an admin may amend the records.
// No-op when checkinRepo isn't wired (legacy/tests).
func (s *DiagnosisService) assertNotLeftYet(ctx context.Context, scheduleID uint) error {
	if s.checkinRepo == nil {
		return nil
	}
	if _, err := s.checkinRepo.WithCtx(ctx).FindByScheduleAndType(scheduleID, "leave"); err == nil {
		return ErrDiagnosisLockedAfterLeave
	}
	return nil
}

// purgePhotos deletes all diagnosis photos (rows + best-effort files) for a
// SchedulePatient. Used when a slot is marked no_show so it never carries stale
// evidence.
func (s *DiagnosisService) purgePhotos(ctx context.Context, spID uint) error {
	photoRepo := s.photoRepo.WithCtx(ctx)
	photos, err := photoRepo.FindBySchedulePatientID(spID)
	if err != nil {
		return err
	}
	for _, p := range photos {
		if err := photoRepo.Delete(p.ID); err != nil {
			return err
		}
		removeUploadedFile(p.PhotoURL)
	}
	return nil
}

// deletePhotoRow deletes the DB row, removes the file best-effort, and reverts
// the slot to pending when it becomes empty.
func (s *DiagnosisService) deletePhotoRow(ctx context.Context, photo *model.DiagnosisPhoto) error {
	photoRepo := s.photoRepo.WithCtx(ctx)
	if err := photoRepo.Delete(photo.ID); err != nil {
		return err
	}
	removeUploadedFile(photo.PhotoURL)

	remaining, err := photoRepo.CountBySchedulePatientID(photo.SchedulePatientID)
	if err != nil {
		return err
	}
	if remaining == 0 {
		// No evidence left — revert to pending so the slot is actionable again.
		return s.spRepo.WithCtx(ctx).UpdateStatus(photo.SchedulePatientID, model.SchedulePatientStatusPending, "")
	}
	return nil
}

// removeUploadedFile best-effort deletes the file backing a "/uploads/<name>"
// URL. Failures are logged but never block the delete operation (e.g. in tests
// the URL has no real file on disk).
func removeUploadedFile(photoURL string) {
	cfg := config.AppConfig
	if cfg == nil || cfg.UploadDir == "" {
		return
	}
	name := strings.TrimPrefix(photoURL, "/uploads/")
	if name == "" || name == photoURL {
		return // not an uploads URL — leave it alone
	}
	// Guard against path traversal: only operate on a bare base filename.
	name = filepath.Base(name)
	path := filepath.Join(cfg.UploadDir, name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("[diagnosis] failed to remove photo file %s: %v", path, err)
	}
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
		PrepaidAmount  int       `gorm:"column:prepaid_amount"`
		ActualAmount   int       `gorm:"column:actual_amount"`
	}
	var rows []row
	err := base.
		Select(`sp.id AS sp_id, sp.schedule_id, sp.start_time AS sp_start, sp.end_time AS sp_end,
			sp.status, sp.no_show_reason, sp.updated_at,
			sp.prepaid_amount, sp.actual_amount,
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
			PrepaidAmount:     r.PrepaidAmount,
			ActualAmount:      r.ActualAmount,
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

var diagnosisResultHeaders = []interface{}{
	"日期", "時段", "地點", "翻譯員", "病人", "電話", "證件類型", "證件號碼",
	"狀態", "未到原因", "預付金額", "實付金額", "照片數",
}

// BuildResultsExcel exports the diagnosis-results overview (per patient, with
// prepaid / actual amounts) as an in-memory xlsx, honouring the same filters as
// ListResults but without pagination.
func (s *DiagnosisService) BuildResultsExcel(ctx context.Context, q dto.DiagnosisResultsQuery) (*excelize.File, error) {
	q.Page = 1
	q.PageSize = 1000000
	resp, err := s.ListResults(ctx, q)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheet := "診斷結果"
	f.NewSheet(sheet)
	f.DeleteSheet("Sheet1")
	for i, h := range diagnosisResultHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for rowIdx, e := range resp.Data {
		statusLabel := e.Status
		switch e.Status {
		case model.SchedulePatientStatusCompleted:
			statusLabel = "已完成"
		case model.SchedulePatientStatusNoShow:
			statusLabel = "未到"
		}
		vals := []interface{}{
			e.Date, e.StartTime + "-" + e.EndTime, e.Location, e.TranslatorName,
			e.PatientName, e.PatientPhone, e.IDType, e.IDNumber,
			statusLabel, e.NoShowReason, e.PrepaidAmount, e.ActualAmount, len(e.DiagnosisPhotos),
		}
		for colIdx, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheet, cell, v)
		}
	}
	return f, nil
}

func indexOfByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

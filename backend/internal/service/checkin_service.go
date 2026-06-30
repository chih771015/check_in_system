package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"gorm.io/gorm"
)

// Sentinel errors returned by CheckinService.
var (
	ErrCheckinNotFound          = errors.New("checkin not found")
	ErrScheduleNotOwned         = errors.New("schedule does not belong to this translator")
	ErrDuplicateCheckin         = errors.New("duplicate checkin type")
	ErrArriveBeforeLeave        = errors.New("must check in (arrive) before checking out (leave)")
	ErrArriveVerifyFailed       = errors.New("failed to verify arrival status")
	ErrCheckinCreate            = errors.New("failed to create checkin record")
	ErrNoFieldsToUpdate         = errors.New("no fields to update")
	ErrCheckoutBlockedByPending = errors.New("cannot check out: some patients are still pending (must upload diagnosis or mark no_show)")
)

// CheckinService handles check-in business logic.
type CheckinService struct {
	checkinRepo  *repository.CheckinRepository
	scheduleRepo *repository.ScheduleRepository
	userRepo     *repository.UserRepository
	geocoding    *GeocodingService
	// Stage 4: when set, leave check-ins are blocked until every
	// SchedulePatient is completed or no_show. Optional so old test fixtures
	// (3-arg with nil) still work.
	spRepo *repository.SchedulePatientRepository
}

// NewCheckinService creates a new CheckinService.
func NewCheckinService(
	checkinRepo *repository.CheckinRepository,
	scheduleRepo *repository.ScheduleRepository,
	userRepo *repository.UserRepository,
	geocoding *GeocodingService,
) *CheckinService {
	return &CheckinService{
		checkinRepo:  checkinRepo,
		scheduleRepo: scheduleRepo,
		userRepo:     userRepo,
		geocoding:    geocoding,
	}
}

// WithSchedulePatientRepo wires the SchedulePatient repo so leave checkins
// can verify every patient slot has been processed.
func (s *CheckinService) WithSchedulePatientRepo(spRepo *repository.SchedulePatientRepository) *CheckinService {
	s.spRepo = spRepo
	return s
}

// Checkin processes a translator's check-in (arrive or leave).
// The context is propagated to reverse-geocoding so the outbound Nominatim
// call appears as a child span of the HTTP request in Jaeger.
func (s *CheckinService) Checkin(
	ctx context.Context,
	translatorID uint,
	scheduleID uint,
	checkinType string,
	lat, lng float64,
	address, selfieURL, envURL string,
	isMakeup bool,
	makeupReason string,
) (*dto.CheckinResponse, error) {
	schRepo := s.scheduleRepo.WithCtx(ctx)
	ckRepo := s.checkinRepo.WithCtx(ctx)
	uRepo := s.userRepo.WithCtx(ctx)

	// Validate schedule exists and belongs to translator
	schedule, err := schRepo.FindByID(scheduleID)
	if err != nil {
		return nil, ErrScheduleNotFound
	}
	if schedule.TranslatorID != translatorID {
		return nil, ErrScheduleNotOwned
	}

	// Check for duplicate checkin type
	existing, err := ckRepo.FindByScheduleAndType(scheduleID, checkinType)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("%w: %s", ErrDuplicateCheckin, checkinType)
	}

	// If leaving, ensure arrival was recorded first
	if checkinType == "leave" {
		_, err := ckRepo.FindByScheduleAndType(scheduleID, "arrive")
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrArriveBeforeLeave
			}
			return nil, ErrArriveVerifyFailed
		}

		// Stage 4: block leave when there are still pending patients.
		// Skipped for makeup checkins (flexible per stage-3 decision B).
		if !isMakeup && s.spRepo != nil {
			sps, err := s.spRepo.WithCtx(ctx).FindByScheduleID(scheduleID)
			if err == nil {
				for _, sp := range sps {
					if sp.Status == "pending" {
						return nil, ErrCheckoutBlockedByPending
					}
				}
			}
		}
	}

	// Auto-detect late / out-of-window checkin.
	// If the current time is past the schedule end time and the caller did not
	// explicitly flag this as a makeup, mark it as makeup automatically so it
	// shows up correctly in admin reports.
	if !isMakeup {
		dateStr := schedule.Date.Format("2006-01-02")
		if endLocal, perr := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+schedule.EndTime, time.Local); perr == nil {
			if time.Now().After(endLocal) {
				isMakeup = true
				if makeupReason == "" {
					makeupReason = "打卡時間超過排班結束時間（系統自動標記）"
				}
			}
		}
	}

	// If address is missing, attempt reverse geocoding. Failures are silently
	// ignored so we never block a checkin over a third-party outage.
	if address == "" && s.geocoding != nil && (lat != 0 || lng != 0) {
		if resolved, err := s.geocoding.ReverseGeocode(ctx, lat, lng); err == nil {
			address = resolved
		}
	}

	// Get translator info
	user, err := uRepo.FindByID(translatorID)
	if err != nil {
		return nil, ErrTranslatorNotFound
	}

	checkin := &model.Checkin{
		ScheduleID:     scheduleID,
		TranslatorID:   translatorID,
		Type:           checkinType,
		CheckinTime:    time.Now(),
		Latitude:       lat,
		Longitude:      lng,
		Address:        address,
		SelfieURL:      selfieURL,
		EnvironmentURL: envURL,
		IsMakeup:       isMakeup,
		MakeupReason:   makeupReason,
	}

	if err := ckRepo.Create(checkin); err != nil {
		return nil, ErrCheckinCreate
	}

	return &dto.CheckinResponse{
		ID:             checkin.ID,
		ScheduleID:     checkin.ScheduleID,
		TranslatorID:   checkin.TranslatorID,
		TranslatorName: user.Name,
		Type:           checkin.Type,
		CheckinTime:    checkin.CheckinTime,
		Latitude:       checkin.Latitude,
		Longitude:      checkin.Longitude,
		Address:        checkin.Address,
		SelfieURL:      checkin.SelfieURL,
		EnvironmentURL: checkin.EnvironmentURL,
		IsMakeup:       checkin.IsMakeup,
		MakeupReason:   checkin.MakeupReason,
		CreatedAt:      checkin.CreatedAt,
	}, nil
}

// AdminUpdateCheckin applies admin-editable fields to an existing checkin.
// Photos and translator/schedule linkage are intentionally not editable.
// AdminUpdateCheckin edits a checkin and returns an audit detail JSON describing
// the before/after state.
func (s *CheckinService) AdminUpdateCheckin(ctx context.Context, id uint, req dto.AdminUpdateCheckinRequest) (string, error) {
	repo := s.checkinRepo.WithCtx(ctx)
	before, err := repo.FindByID(id)
	if err != nil {
		return "", ErrCheckinNotFound
	}

	fields := map[string]any{}
	if req.CheckinTime != nil {
		fields["checkin_time"] = *req.CheckinTime
	}
	if req.Address != nil {
		fields["address"] = *req.Address
	}
	if req.MakeupReason != nil {
		fields["makeup_reason"] = *req.MakeupReason
	}
	if len(fields) == 0 {
		return "", ErrNoFieldsToUpdate
	}
	beforeSnap := snapshotCheckin(before)
	if err := repo.UpdateFields(id, fields); err != nil {
		return "", err
	}
	after, err := repo.FindByID(id)
	if err != nil {
		return "", err
	}
	return auditDetailJSON(beforeSnap, snapshotCheckin(after)), nil
}

// AdminDeleteCheckin permanently removes a checkin record and returns an audit
// detail JSON containing a snapshot of the deleted record.
func (s *CheckinService) AdminDeleteCheckin(ctx context.Context, id uint) (string, error) {
	repo := s.checkinRepo.WithCtx(ctx)
	before, err := repo.FindByID(id)
	if err != nil {
		return "", ErrCheckinNotFound
	}
	if err := repo.Delete(id); err != nil {
		return "", err
	}
	return auditDetailJSON(snapshotCheckin(before), nil), nil
}

// MyHistory returns the translator's own checkin history with optional filters.
func (s *CheckinService) MyHistory(ctx context.Context, translatorID uint, dateFrom, dateTo string) ([]dto.CheckinResponse, error) {
	checkins, _, err := s.checkinRepo.WithCtx(ctx).ListAll(repository.ListAllParams{
		DateFrom:     dateFrom,
		DateTo:       dateTo,
		TranslatorID: translatorID,
	})
	if err != nil {
		return nil, err
	}
	user, _ := s.userRepo.WithCtx(ctx).FindByID(translatorID)
	name := ""
	if user != nil {
		name = user.Name
	}
	results := make([]dto.CheckinResponse, 0, len(checkins))
	for _, c := range checkins {
		results = append(results, dto.CheckinResponse{
			ID:             c.ID,
			ScheduleID:     c.ScheduleID,
			TranslatorID:   c.TranslatorID,
			TranslatorName: name,
			Type:           c.Type,
			CheckinTime:    c.CheckinTime,
			Latitude:       c.Latitude,
			Longitude:      c.Longitude,
			Address:        c.Address,
			SelfieURL:      c.SelfieURL,
			EnvironmentURL: c.EnvironmentURL,
			IsMakeup:       c.IsMakeup,
			MakeupReason:   c.MakeupReason,
			CreatedAt:      c.CreatedAt,
		})
	}
	return results, nil
}

// MyStats returns aggregate stats for a translator within the given date range.
type CheckinStats struct {
	Total       int `json:"total"`
	ArriveCount int `json:"arriveCount"`
	LeaveCount  int `json:"leaveCount"`
	MakeupCount int `json:"makeupCount"`
	// MakeupArriveCount is the subset of arrives that are makeup. The frontend
	// annotates the "arrive" card with it ("含 N 筆補打卡") so arrive vs
	// onTime+late no longer looks like it's missing one.
	MakeupArriveCount int `json:"makeupArriveCount"`
	OnTimeCount       int `json:"onTimeCount"`
	LateCount         int `json:"lateCount"`
}

// MyStats computes aggregate checkin stats for a translator.
func (s *CheckinService) MyStats(ctx context.Context, translatorID uint, dateFrom, dateTo string) (*CheckinStats, error) {
	ckRepo := s.checkinRepo.WithCtx(ctx)
	schRepo := s.scheduleRepo.WithCtx(ctx)
	checkins, _, err := ckRepo.ListAll(repository.ListAllParams{
		DateFrom:     dateFrom,
		DateTo:       dateTo,
		TranslatorID: translatorID,
	})
	if err != nil {
		return nil, err
	}
	stats := &CheckinStats{Total: len(checkins)}
	for _, c := range checkins {
		switch c.Type {
		case "arrive":
			stats.ArriveCount++
			// Punctuality is only meaningful for a real, in-the-moment arrive.
			// A makeup arrive carries the backfill time (time.Now() when it was
			// logged), not the real arrival, so comparing it to the schedule
			// start would wrongly flag it as late — skip it entirely here; it is
			// counted under MakeupCount below instead.
			if c.IsMakeup {
				stats.MakeupArriveCount++
				break
			}
			// Compare with schedule start time to detect late arrival.
			if sch, err := schRepo.FindByID(c.ScheduleID); err == nil {
				dateStr := sch.Date.Format("2006-01-02")
				if startLocal, perr := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+sch.StartTime, time.Local); perr == nil {
					if c.CheckinTime.After(startLocal.Add(5 * time.Minute)) {
						stats.LateCount++
					} else {
						stats.OnTimeCount++
					}
				}
			}
		case "leave":
			stats.LeaveCount++
		}
		if c.IsMakeup {
			stats.MakeupCount++
		}
	}
	return stats, nil
}

// AdminListParams mirrors repository.ListAllParams for service layer.
// PageSize <= 0 means "no pagination" (the export path relies on this to pull
// every matching row).
type AdminListParams struct {
	DateFrom     string
	DateTo       string
	TranslatorID uint
	CheckinType  string
	IsMakeup     *bool
	Page         int
	PageSize     int
}

// AdminList returns one page of checkins matching the filters plus the total
// matching count. With PageSize <= 0 it returns every row (export path).
func (s *CheckinService) AdminList(ctx context.Context, params AdminListParams) ([]dto.CheckinResponse, int64, error) {
	checkins, total, err := s.checkinRepo.WithCtx(ctx).ListAll(repository.ListAllParams{
		DateFrom:     params.DateFrom,
		DateTo:       params.DateTo,
		TranslatorID: params.TranslatorID,
		CheckinType:  params.CheckinType,
		IsMakeup:     params.IsMakeup,
		Page:         params.Page,
		PageSize:     params.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	uRepo := s.userRepo.WithCtx(ctx)
	results := make([]dto.CheckinResponse, 0, len(checkins))
	for _, c := range checkins {
		user, err := uRepo.FindByID(c.TranslatorID)
		translatorName := ""
		if err == nil {
			translatorName = user.Name
		}
		results = append(results, dto.CheckinResponse{
			ID:             c.ID,
			ScheduleID:     c.ScheduleID,
			TranslatorID:   c.TranslatorID,
			TranslatorName: translatorName,
			Type:           c.Type,
			CheckinTime:    c.CheckinTime,
			Latitude:       c.Latitude,
			Longitude:      c.Longitude,
			Address:        c.Address,
			SelfieURL:      c.SelfieURL,
			EnvironmentURL: c.EnvironmentURL,
			IsMakeup:       c.IsMakeup,
			MakeupReason:   c.MakeupReason,
			CreatedAt:      c.CreatedAt,
		})
	}
	return results, total, nil
}

package service

import (
	"context"
	"time"

	"translator-checkin/internal/repository"
)

// StatsService provides aggregate figures for the admin dashboard banner.
type StatsService struct {
	spRepo *repository.SchedulePatientRepository
}

// NewStatsService creates a new StatsService.
func NewStatsService(spRepo *repository.SchedulePatientRepository) *StatsService {
	return &StatsService{spRepo: spRepo}
}

// monthRange returns the current month as a half-open [from, to) pair of
// YYYY-MM-DD strings plus a YYYY-MM label. Pulled out as a pure function of
// `now` so the month boundary logic (incl. December → next year) is testable.
func monthRange(now time.Time) (from, to, label string) {
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	next := first.AddDate(0, 1, 0)
	return first.Format("2006-01-02"), next.Format("2006-01-02"), first.Format("2006-01")
}

// CurrentMonthActualTotal returns the actual-paid total across all patients for
// the current calendar month (by schedule date), with the YYYY-MM label.
func (s *StatsService) CurrentMonthActualTotal(ctx context.Context) (string, int64, error) {
	from, to, label := monthRange(time.Now())
	total, err := s.spRepo.WithCtx(ctx).SumActualByDateRange(from, to)
	if err != nil {
		return label, 0, err
	}
	return label, total, nil
}

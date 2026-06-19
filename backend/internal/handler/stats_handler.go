package handler

import (
	"net/http"

	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// StatsHandler exposes admin dashboard aggregate figures.
type StatsHandler struct {
	statsService *service.StatsService
}

// NewStatsHandler creates a new StatsHandler.
func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// MonthlyTotal handles GET /api/admin/stats/monthly-total and returns the
// actual-paid total across all patients for the current calendar month.
func (h *StatsHandler) MonthlyTotal(c *gin.Context) {
	label, total, err := h.statsService.CurrentMonthActualTotal(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"yearMonth": label, "total": total})
}

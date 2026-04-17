package handler

import (
	"net/http"
	"time"

	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// ExportScheduleHandler handles export schedule endpoints.
type ExportScheduleHandler struct {
	repo          *repository.ExportScheduleRepository
	exportService *service.ExportService
}

// NewExportScheduleHandler creates a new ExportScheduleHandler.
func NewExportScheduleHandler(repo *repository.ExportScheduleRepository, exportService *service.ExportService) *ExportScheduleHandler {
	return &ExportScheduleHandler{repo: repo, exportService: exportService}
}

// GetExportSchedule handles GET /api/admin/export/schedule
func (h *ExportScheduleHandler) GetExportSchedule(c *gin.Context) {
	adminID := c.GetUint("userID")
	es, err := h.repo.WithCtx(c.Request.Context()).FindByAdmin(adminID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": map[string]interface{}{
		"id":         es.ID,
		"frequency":  es.Frequency,
		"dayOfMonth": es.DayOfMonth,
		"format":     es.Format,
		"emailTo":    es.EmailTo,
		"enabled":    es.Enabled,
		"lastRunAt":  es.LastRunAt,
	}})
}

// UpsertExportSchedule handles POST /api/admin/export/schedule
func (h *ExportScheduleHandler) UpsertExportSchedule(c *gin.Context) {
	adminID := c.GetUint("userID")
	var req struct {
		Frequency  string `json:"frequency" binding:"required"`
		DayOfMonth int    `json:"dayOfMonth" binding:"required,min=1,max=28"`
		Format     string `json:"format" binding:"required,oneof=excel google_sheet"`
		EmailTo    string `json:"emailTo"`
		Enabled    bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	es := &model.ExportSchedule{
		AdminID:    adminID,
		Frequency:  req.Frequency,
		DayOfMonth: req.DayOfMonth,
		Format:     req.Format,
		EmailTo:    req.EmailTo,
		Enabled:    req.Enabled,
		UpdatedAt:  now,
	}
	if err := h.repo.WithCtx(c.Request.Context()).Upsert(es); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Export schedule saved"})
}

// RunExportNow handles POST /api/admin/export/schedule/run.
// Triggers the same export logic the cron uses, immediately, for the calling
// admin. Useful for verifying the configuration without waiting until the
// scheduled day.
func (h *ExportScheduleHandler) RunExportNow(c *gin.Context) {
	adminID := c.GetUint("userID")
	result, err := h.exportService.RunExportForAdmin(c.Request.Context(), adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Export executed successfully",
		"result":  result,
	})
}

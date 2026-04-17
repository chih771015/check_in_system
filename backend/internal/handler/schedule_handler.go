package handler

import (
	"net/http"
	"strconv"
	"strings"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// ScheduleHandler handles schedule-related endpoints.
type ScheduleHandler struct {
	scheduleService *service.ScheduleService
	auditService    *service.AuditService
}

// NewScheduleHandler creates a new ScheduleHandler.
func NewScheduleHandler(scheduleService *service.ScheduleService, auditService *service.AuditService) *ScheduleHandler {
	return &ScheduleHandler{scheduleService: scheduleService, auditService: auditService}
}

// AdminListSchedules handles GET /api/admin/schedules.
func (h *ScheduleHandler) AdminListSchedules(c *gin.Context) {
	var query dto.ScheduleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var translatorID uint
	if query.TranslatorID != "" {
		id, err := strconv.ParseUint(query.TranslatorID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid translator_id"})
			return
		}
		translatorID = uint(id)
	}

	schedules, err := h.scheduleService.List(c.Request.Context(), translatorID, query.DateFrom, query.DateTo, query.Location)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": schedules})
}

// AdminCreateSchedule handles POST /api/admin/schedules.
func (h *ScheduleHandler) AdminCreateSchedule(c *gin.Context) {
	var req dto.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.scheduleService.Create(ctx, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "create_schedule", "schedule", 0, "")

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// AdminUpdateSchedule handles PUT /api/admin/schedules/:id.
func (h *ScheduleHandler) AdminUpdateSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	var req dto.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.scheduleService.Update(ctx, uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "update_schedule", "schedule", uint(id), "")

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// AdminDeleteSchedule handles DELETE /api/admin/schedules/:id.
func (h *ScheduleHandler) AdminDeleteSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	ctx := c.Request.Context()
	if err := h.scheduleService.Delete(ctx, uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "delete_schedule", "schedule", uint(id), "")

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted successfully"})
}

// AdminDeleteScheduleGroup handles DELETE /api/admin/schedules/:id/group.
// Removes every schedule that shares the recurrence group of the given id.
func (h *ScheduleHandler) AdminDeleteScheduleGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	ctx := c.Request.Context()
	count, err := h.scheduleService.DeleteRecurrenceGroup(ctx, uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "delete_schedule_group", "schedule", uint(id), "")

	c.JSON(http.StatusOK, gin.H{
		"message": "Schedule group deleted successfully",
		"deleted": count,
	})
}

// AdminImportSchedules handles POST /api/admin/schedules/import.
// Accepts a multipart Excel file with a header row:
//   translatorId | date | startTime | endTime | location | patientName | note
func (h *ScheduleHandler) AdminImportSchedules(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open file"})
		return
	}
	defer f.Close()

	xl, err := excelize.OpenReader(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid excel file: " + err.Error()})
		return
	}
	defer xl.Close()

	sheet := xl.GetSheetName(0)
	xrows, err := xl.GetRows(sheet)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read rows"})
		return
	}

	var rows []service.ScheduleImportRow
	for i, r := range xrows {
		if i == 0 {
			continue // skip header
		}
		if len(r) == 0 {
			continue
		}
		row := service.ScheduleImportRow{RowNumber: i + 1}
		get := func(idx int) string {
			if idx < len(r) {
				return strings.TrimSpace(r[idx])
			}
			return ""
		}
		tidStr := get(0)
		if tidStr == "" {
			row.Error = "translatorId is empty"
			rows = append(rows, row)
			continue
		}
		tid, perr := strconv.ParseUint(tidStr, 10, 32)
		if perr != nil {
			row.Error = "translatorId must be numeric"
			rows = append(rows, row)
			continue
		}
		row.TranslatorID = uint(tid)
		row.Date = get(1)
		row.StartTime = get(2)
		row.EndTime = get(3)
		row.Location = get(4)
		row.PatientName = get(5)
		row.Note = get(6)
		if row.Date == "" || row.StartTime == "" || row.EndTime == "" || row.Location == "" || row.PatientName == "" {
			row.Error = "missing required field"
		}
		rows = append(rows, row)
	}

	ctx := c.Request.Context()
	success, failed := h.scheduleService.BatchImportSchedules(ctx, rows)

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "import_schedules", "schedule", 0, "imported via excel")

	c.JSON(http.StatusOK, gin.H{
		"success": success,
		"failed":  failed,
		"total":   len(rows),
	})
}

// MySchedules handles GET /api/schedules for translators.
func (h *ScheduleHandler) MySchedules(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	schedules, err := h.scheduleService.ListForTranslator(c.Request.Context(), userID.(uint), dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": schedules})
}

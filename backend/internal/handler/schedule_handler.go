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
		respondBadRequest(c, err)
		return
	}

	var translatorID uint
	if query.TranslatorID != "" {
		id, err := strconv.ParseUint(query.TranslatorID, 10, 32)
		if err != nil {
			respondCode(c, http.StatusBadRequest, dto.CodeInvalidTranslatorID, "Invalid translator_id")
			return
		}
		translatorID = uint(id)
	}

	schedules, err := h.scheduleService.List(c.Request.Context(), translatorID, query.DateFrom, query.DateTo, query.Location)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": schedules})
}

// AdminCreateSchedule handles POST /api/admin/schedules.
func (h *ScheduleHandler) AdminCreateSchedule(c *gin.Context) {
	var req dto.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	ctx := c.Request.Context()
	resp, err := h.scheduleService.Create(ctx, req)
	if err != nil {
		respondError(c, err)
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
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidScheduleID, "Invalid schedule ID")
		return
	}

	var req dto.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	ctx := c.Request.Context()
	resp, err := h.scheduleService.Update(ctx, uint(id), req)
	if err != nil {
		respondError(c, err)
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
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidScheduleID, "Invalid schedule ID")
		return
	}

	ctx := c.Request.Context()
	if err := h.scheduleService.Delete(ctx, uint(id)); err != nil {
		respondError(c, err)
		return
	}

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "delete_schedule", "schedule", uint(id), "")

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted successfully"})
}

// AdminDeleteScheduleGroup handles DELETE /api/admin/schedules/:id/group.
func (h *ScheduleHandler) AdminDeleteScheduleGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidScheduleID, "Invalid schedule ID")
		return
	}

	ctx := c.Request.Context()
	count, err := h.scheduleService.DeleteRecurrenceGroup(ctx, uint(id))
	if err != nil {
		respondError(c, err)
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
func (h *ScheduleHandler) AdminImportSchedules(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeFileRequired, "file is required")
		return
	}
	f, err := file.Open()
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeFileOpenFailed, "failed to open file")
		return
	}
	defer f.Close()

	xl, err := excelize.OpenReader(f)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidExcel, "invalid excel file: "+err.Error())
		return
	}
	defer xl.Close()

	sheet := xl.GetSheetName(0)
	xrows, err := xl.GetRows(sheet)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeReadRowsFailed, "failed to read rows")
		return
	}

	// Stage-3 flat format columns:
	//   A=Code | B=TranslatorID | C=Date | D=OverallStart | E=OverallEnd |
	//   F=Location | G=PatientID | H=PatientStart | I=PatientEnd | J=Note(optional)
	var rows []service.ScheduleImportRowV2
	parseFailed := []service.ScheduleImportRowV2{}
	for i, r := range xrows {
		if i == 0 {
			continue
		}
		if len(r) == 0 {
			continue
		}
		row := service.ScheduleImportRowV2{RowNumber: i + 1}
		get := func(idx int) string {
			if idx < len(r) {
				return strings.TrimSpace(r[idx])
			}
			return ""
		}
		row.Code = get(0)
		tidStr := get(1)
		if tidStr != "" {
			if tid, perr := strconv.ParseUint(tidStr, 10, 32); perr == nil {
				row.TranslatorID = uint(tid)
			} else {
				row.Error = "translatorId must be numeric"
				parseFailed = append(parseFailed, row)
				continue
			}
		}
		row.Date = get(2)
		row.OverallStart = get(3)
		row.OverallEnd = get(4)
		row.Location = get(5)
		pidStr := get(6)
		if pidStr != "" {
			if pid, perr := strconv.ParseUint(pidStr, 10, 32); perr == nil {
				row.PatientID = uint(pid)
			} else {
				row.Error = "patientId must be numeric"
				parseFailed = append(parseFailed, row)
				continue
			}
		}
		row.PatientStart = get(7)
		row.PatientEnd = get(8)
		row.Note = get(9)
		if row.Code == "" || row.TranslatorID == 0 || row.Date == "" ||
			row.OverallStart == "" || row.OverallEnd == "" || row.Location == "" ||
			row.PatientID == 0 || row.PatientStart == "" || row.PatientEnd == "" {
			row.Error = "missing required field"
			parseFailed = append(parseFailed, row)
			continue
		}
		rows = append(rows, row)
	}

	ctx := c.Request.Context()
	result, _ := h.scheduleService.BatchImportSchedulesV2(ctx, rows)
	result.Failed = append(parseFailed, result.Failed...)

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "import_schedules", "schedule", 0, "imported via excel (v2 multi-patient)")

	c.JSON(http.StatusOK, gin.H{
		"successSchedules": result.SuccessSchedules,
		"successPatients":  result.SuccessPatients,
		"failed":           result.Failed,
		"total":            len(xrows) - 1,
	})
}

// MySchedules handles GET /api/schedules for translators.
func (h *ScheduleHandler) MySchedules(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		respondCode(c, http.StatusUnauthorized, dto.CodeUserContextMissing, "User not found in context")
		return
	}

	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	schedules, err := h.scheduleService.ListForTranslator(c.Request.Context(), userID.(uint), dateFrom, dateTo)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": schedules})
}

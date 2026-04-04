package handler

import (
	"net/http"
	"strconv"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// ScheduleHandler handles schedule-related endpoints.
type ScheduleHandler struct {
	scheduleService *service.ScheduleService
}

// NewScheduleHandler creates a new ScheduleHandler.
func NewScheduleHandler(scheduleService *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{scheduleService: scheduleService}
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

	schedules, err := h.scheduleService.List(translatorID, query.DateFrom, query.DateTo, query.Location)
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

	resp, err := h.scheduleService.Create(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	resp, err := h.scheduleService.Update(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// AdminDeleteSchedule handles DELETE /api/admin/schedules/:id.
func (h *ScheduleHandler) AdminDeleteSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	if err := h.scheduleService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted successfully"})
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

	schedules, err := h.scheduleService.ListForTranslator(userID.(uint), dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": schedules})
}

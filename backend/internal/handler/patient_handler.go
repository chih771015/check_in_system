package handler

import (
	"errors"
	"net/http"
	"strconv"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// PatientHandler exposes patient CRUD endpoints to admins and a trimmed list
// endpoint to translators.
type PatientHandler struct {
	patientService *service.PatientService
	auditService   *service.AuditService
}

// NewPatientHandler creates a new PatientHandler.
func NewPatientHandler(patientService *service.PatientService, auditService *service.AuditService) *PatientHandler {
	return &PatientHandler{patientService: patientService, auditService: auditService}
}

// toPatientResponse maps a model.Patient to the admin response DTO.
func toPatientResponse(p *model.Patient) dto.PatientResponse {
	return dto.PatientResponse{
		ID:        p.ID,
		Name:      p.Name,
		Phone:     p.Phone,
		IDType:    p.IDType,
		IDNumber:  p.IDNumber,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// toTranslatorPatientResponse maps a model.Patient to the trimmed translator
// view (no timestamps).
func toTranslatorPatientResponse(p *model.Patient) dto.TranslatorPatientResponse {
	return dto.TranslatorPatientResponse{
		ID:       p.ID,
		Name:     p.Name,
		Phone:    p.Phone,
		IDType:   p.IDType,
		IDNumber: p.IDNumber,
	}
}

// ListPatients handles GET /api/admin/patients
func (h *PatientHandler) ListPatients(c *gin.Context) {
	var q dto.PatientListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	patients, total, err := h.patientService.List(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	data := make([]dto.PatientResponse, len(patients))
	for i := range patients {
		data[i] = toPatientResponse(&patients[i])
	}
	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	c.JSON(http.StatusOK, dto.PatientListResponse{
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// CreatePatient handles POST /api/admin/patients
func (h *PatientHandler) CreatePatient(c *gin.Context) {
	var req dto.CreatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	patient, err := h.patientService.Create(ctx, req)
	if err != nil {
		if errors.Is(err, service.ErrPatientDuplicate) {
			c.JSON(http.StatusConflict, gin.H{"error": "病人資料重複（相同 ID 類型與號碼）"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	requesterID := c.GetUint("userID")
	h.auditService.Log(ctx, requesterID, "create_patient", "patient", patient.ID, "name="+patient.Name)
	c.JSON(http.StatusCreated, gin.H{"data": toPatientResponse(patient)})
}

// UpdatePatient handles PUT /api/admin/patients/:id
func (h *PatientHandler) UpdatePatient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的病人 ID"})
		return
	}
	var req dto.UpdatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	patient, err := h.patientService.Update(ctx, uint(id), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPatientDuplicate):
			c.JSON(http.StatusConflict, gin.H{"error": "病人資料重複（相同 ID 類型與號碼）"})
		case errors.Is(err, service.ErrPatientNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到此病人"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	requesterID := c.GetUint("userID")
	h.auditService.Log(ctx, requesterID, "update_patient", "patient", patient.ID, "name="+patient.Name)
	c.JSON(http.StatusOK, gin.H{"data": toPatientResponse(patient)})
}

// DeletePatient handles DELETE /api/admin/patients/:id
func (h *PatientHandler) DeletePatient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的病人 ID"})
		return
	}
	ctx := c.Request.Context()
	if err := h.patientService.Delete(ctx, uint(id)); err != nil {
		if errors.Is(err, service.ErrPatientNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到此病人"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	requesterID := c.GetUint("userID")
	h.auditService.Log(ctx, requesterID, "delete_patient", "patient", uint(id), "")
	c.JSON(http.StatusOK, gin.H{"message": "病人資料已刪除"})
}

// GetPatientHistory handles GET /api/admin/patients/:id/history
// Stage 2 returns an empty history list; stage 4 will fill it in.
func (h *PatientHandler) GetPatientHistory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的病人 ID"})
		return
	}
	resp, err := h.patientService.GetHistory(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, service.ErrPatientNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到此病人"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListPatientsForTranslator handles GET /api/patients
// Returns the trimmed translator view (no timestamps).
// TODO(stage 3): restrict to patients tied to schedules owned by the caller.
func (h *PatientHandler) ListPatientsForTranslator(c *gin.Context) {
	var q dto.PatientListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	translatorID := c.GetUint("userID")
	patients, total, err := h.patientService.ListForTranslator(c.Request.Context(), translatorID, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	data := make([]dto.TranslatorPatientResponse, len(patients))
	for i := range patients {
		data[i] = toTranslatorPatientResponse(&patients[i])
	}
	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	c.JSON(http.StatusOK, gin.H{
		"data":     data,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

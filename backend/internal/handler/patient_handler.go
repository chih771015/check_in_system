package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
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

func toTranslatorPatientResponse(p *model.Patient) dto.TranslatorPatientResponse {
	return dto.TranslatorPatientResponse{
		ID:       p.ID,
		Name:     p.Name,
		Phone:    p.Phone,
		IDType:   p.IDType,
		IDNumber: p.IDNumber,
	}
}

// ExportPatients handles GET /api/admin/export/patients — all patients as xlsx.
func (h *PatientHandler) ExportPatients(c *gin.Context) {
	f, err := h.patientService.BuildExcel(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="patients.xlsx"`)
	if err := f.Write(c.Writer); err != nil {
		respondCode(c, http.StatusInternalServerError, dto.CodeExportFailed, "Failed to generate Excel")
	}
}

// DownloadPatientTemplate handles GET /api/admin/export/patients-template.
func (h *PatientHandler) DownloadPatientTemplate(c *gin.Context) {
	f := service.BuildPatientTemplate()
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="patients_template.xlsx"`)
	if err := f.Write(c.Writer); err != nil {
		respondCode(c, http.StatusInternalServerError, dto.CodeExportFailed, "Failed to generate template")
	}
}

// ImportPatients handles POST /api/admin/patients/import (multipart xlsx).
// Columns: A=Name | B=Phone | C=IDType(passport/hn/unid) | D=IDNumber.
// Duplicates / invalid rows are skipped and reported; valid rows created.
func (h *PatientHandler) ImportPatients(c *gin.Context) {
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

	xrows, err := xl.GetRows(xl.GetSheetName(0))
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeReadRowsFailed, "failed to read rows")
		return
	}

	rows := make([]service.PatientImportRow, 0, len(xrows))
	for i, r := range xrows {
		if i == 0 {
			continue // skip header row
		}
		rows = append(rows, service.PatientImportRow{
			Name:     cellAt(r, 0),
			Phone:    cellAt(r, 1),
			IDType:   cellAt(r, 2),
			IDNumber: cellAt(r, 3),
		})
	}

	ctx := c.Request.Context()
	res := h.patientService.ImportPatients(ctx, rows)
	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "import_patients", "patient", 0,
		fmt.Sprintf("created=%d skipped=%d", res.Created, res.Skipped))
	c.JSON(http.StatusOK, res)
}

// cellAt safely reads column idx from a sheet row (missing column → "").
func cellAt(row []string, idx int) string {
	if idx < len(row) {
		return row[idx]
	}
	return ""
}

// ListPatients handles GET /api/admin/patients
func (h *PatientHandler) ListPatients(c *gin.Context) {
	var q dto.PatientListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		respondBadRequest(c, err)
		return
	}
	patients, total, err := h.patientService.List(c.Request.Context(), q)
	if err != nil {
		respondError(c, err)
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
		respondBadRequest(c, err)
		return
	}
	ctx := c.Request.Context()
	patient, err := h.patientService.Create(ctx, req)
	if err != nil {
		respondError(c, err)
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
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidPatientID, "Invalid patient ID")
		return
	}
	var req dto.UpdatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}
	ctx := c.Request.Context()
	patient, err := h.patientService.Update(ctx, uint(id), req)
	if err != nil {
		respondError(c, err)
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
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidPatientID, "Invalid patient ID")
		return
	}
	ctx := c.Request.Context()
	if err := h.patientService.Delete(ctx, uint(id)); err != nil {
		respondError(c, err)
		return
	}
	requesterID := c.GetUint("userID")
	h.auditService.Log(ctx, requesterID, "delete_patient", "patient", uint(id), "")
	c.JSON(http.StatusOK, gin.H{"message": "Patient deleted"})
}

// GetPatientHistory handles GET /api/admin/patients/:id/history
// Stage 2 returns an empty history list; stage 4 will fill it in.
func (h *PatientHandler) GetPatientHistory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidPatientID, "Invalid patient ID")
		return
	}
	resp, err := h.patientService.GetHistory(c.Request.Context(), uint(id))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListPatientsForTranslator handles GET /api/patients
// TODO(stage 3): restrict to patients tied to schedules owned by the caller.
func (h *PatientHandler) ListPatientsForTranslator(c *gin.Context) {
	var q dto.PatientListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		respondBadRequest(c, err)
		return
	}
	translatorID := c.GetUint("userID")
	patients, total, err := h.patientService.ListForTranslator(c.Request.Context(), translatorID, q)
	if err != nil {
		respondError(c, err)
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

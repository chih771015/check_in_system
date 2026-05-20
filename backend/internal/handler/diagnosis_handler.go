package handler

import (
	"net/http"
	"strconv"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// DiagnosisHandler exposes diagnosis photo upload + no-show endpoints for
// translators (own patients) and admins (surrogate).
type DiagnosisHandler struct {
	diagService  *service.DiagnosisService
	auditService *service.AuditService
}

// NewDiagnosisHandler creates a new DiagnosisHandler.
func NewDiagnosisHandler(diagService *service.DiagnosisService, auditService *service.AuditService) *DiagnosisHandler {
	return &DiagnosisHandler{diagService: diagService, auditService: auditService}
}

// UploadDiagnosis handles POST /api/checkins/diagnosis (multipart form).
//
// Form fields:
//   - schedulePatientId (uint, required)
//   - photo (file, repeatable up to 3)
func (h *DiagnosisHandler) UploadDiagnosis(c *gin.Context) {
	spIDStr := c.PostForm("schedulePatientId")
	spID, err := strconv.ParseUint(spIDStr, 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeSchedulePatientNotFound, "Invalid schedulePatientId")
		return
	}

	translatorID := c.GetUint("userID")

	form, err := c.MultipartForm()
	if err != nil {
		respondBadRequest(c, err)
		return
	}
	files := form.File["photo"]
	if len(files) == 0 {
		respondCode(c, http.StatusBadRequest, dto.CodeBadRequest, "at least one photo is required")
		return
	}
	urls := make([]string, 0, len(files))
	for i, fh := range files {
		_ = i
		url, err := saveMultipartFile(c, fh, "diagnosis")
		if err != nil {
			respondBadRequest(c, err)
			return
		}
		urls = append(urls, url)
	}

	if err := h.diagService.UploadDiagnosis(c.Request.Context(), translatorID, uint(spID), urls); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Diagnosis uploaded", "photoUrls": urls})
}

// MarkNoShow handles POST /api/checkins/no-show.
func (h *DiagnosisHandler) MarkNoShow(c *gin.Context) {
	var req dto.MarkNoShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}
	translatorID := c.GetUint("userID")
	if err := h.diagService.MarkNoShow(c.Request.Context(), translatorID, req.SchedulePatientID, req.Reason); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Marked as no-show"})
}

// AdminUploadDiagnosis handles POST /api/admin/diagnosis (admin surrogate).
func (h *DiagnosisHandler) AdminUploadDiagnosis(c *gin.Context) {
	spIDStr := c.PostForm("schedulePatientId")
	spID, err := strconv.ParseUint(spIDStr, 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeSchedulePatientNotFound, "Invalid schedulePatientId")
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		respondBadRequest(c, err)
		return
	}
	files := form.File["photo"]
	if len(files) == 0 {
		respondCode(c, http.StatusBadRequest, dto.CodeBadRequest, "at least one photo is required")
		return
	}
	urls := make([]string, 0, len(files))
	for _, fh := range files {
		url, err := saveMultipartFile(c, fh, "diagnosis")
		if err != nil {
			respondBadRequest(c, err)
			return
		}
		urls = append(urls, url)
	}

	ctx := c.Request.Context()
	if err := h.diagService.AdminUploadDiagnosis(ctx, uint(spID), urls); err != nil {
		respondError(c, err)
		return
	}
	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "admin_upload_diagnosis", "schedule_patient", uint(spID), "")
	c.JSON(http.StatusOK, gin.H{"message": "Diagnosis uploaded (admin)"})
}

// AdminMarkNoShow handles POST /api/admin/no-show (admin surrogate).
func (h *DiagnosisHandler) AdminMarkNoShow(c *gin.Context) {
	var req dto.MarkNoShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}
	ctx := c.Request.Context()
	if err := h.diagService.AdminMarkNoShow(ctx, req.SchedulePatientID, req.Reason); err != nil {
		respondError(c, err)
		return
	}
	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "admin_mark_no_show", "schedule_patient", req.SchedulePatientID, req.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "Marked as no-show (admin)"})
}

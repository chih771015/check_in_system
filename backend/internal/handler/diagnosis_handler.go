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

// ListMyPhotos handles GET /api/checkins/diagnosis/photos?schedulePatientId=ID
// — returns the diagnosis photos (with IDs) for a SchedulePatient owned by the
// requesting translator, so the manage modal can delete specific photos.
func (h *DiagnosisHandler) ListMyPhotos(c *gin.Context) {
	spID, err := strconv.ParseUint(c.Query("schedulePatientId"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeSchedulePatientNotFound, "Invalid schedulePatientId")
		return
	}
	translatorID := c.GetUint("userID")
	items, err := h.diagService.ListPhotoItems(c.Request.Context(), translatorID, uint(spID))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": items})
}

// DeleteMyPhoto handles DELETE /api/checkins/diagnosis/photos/:photoId
// — removes one diagnosis photo owned by the requesting translator.
func (h *DiagnosisHandler) DeleteMyPhoto(c *gin.Context) {
	photoID, err := strconv.ParseUint(c.Param("photoId"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeDiagnosisPhotoNotFound, "Invalid photoId")
		return
	}
	translatorID := c.GetUint("userID")
	if err := h.diagService.DeletePhoto(c.Request.Context(), translatorID, uint(photoID)); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Diagnosis photo deleted"})
}

// SetActualAmount handles POST /api/checkins/diagnosis/amount — translator sets
// the actual paid amount for one of their own SchedulePatients.
func (h *DiagnosisHandler) SetActualAmount(c *gin.Context) {
	var req dto.SetActualAmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}
	translatorID := c.GetUint("userID")
	if err := h.diagService.SetActualAmount(c.Request.Context(), translatorID, req.SchedulePatientID, req.ActualAmount); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Actual amount saved"})
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

// AdminGetSchedulePatientPhotos handles GET /api/admin/schedule-patients/:id/photos
// — returns the photo URLs attached to one SchedulePatient slot.
func (h *DiagnosisHandler) AdminGetSchedulePatientPhotos(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeSchedulePatientNotFound, "Invalid schedulePatientId")
		return
	}
	urls, err := h.diagService.GetPhotos(c.Request.Context(), uint(id))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": urls})
}

// AdminListPhotoItems handles GET /api/admin/diagnosis/photos?schedulePatientId=ID
// — like ListMyPhotos but admin-surrogate (no ownership check); returns IDs so
// the admin manage modal can delete specific photos.
func (h *DiagnosisHandler) AdminListPhotoItems(c *gin.Context) {
	spID, err := strconv.ParseUint(c.Query("schedulePatientId"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeSchedulePatientNotFound, "Invalid schedulePatientId")
		return
	}
	items, err := h.diagService.AdminListPhotoItems(c.Request.Context(), uint(spID))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": items})
}

// AdminDeletePhoto handles DELETE /api/admin/diagnosis/photos/:photoId
// — admin-surrogate delete of one diagnosis photo.
func (h *DiagnosisHandler) AdminDeletePhoto(c *gin.Context) {
	photoID, err := strconv.ParseUint(c.Param("photoId"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeDiagnosisPhotoNotFound, "Invalid photoId")
		return
	}
	ctx := c.Request.Context()
	if err := h.diagService.AdminDeletePhoto(ctx, uint(photoID)); err != nil {
		respondError(c, err)
		return
	}
	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "admin_delete_diagnosis_photo", "diagnosis_photo", uint(photoID), "")
	c.JSON(http.StatusOK, gin.H{"message": "Diagnosis photo deleted (admin)"})
}

// AdminSetActualAmount handles POST /api/admin/diagnosis/amount — admin sets the
// actual paid amount for any SchedulePatient (no ownership check).
func (h *DiagnosisHandler) AdminSetActualAmount(c *gin.Context) {
	var req dto.SetActualAmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}
	ctx := c.Request.Context()
	if err := h.diagService.AdminSetActualAmount(ctx, req.SchedulePatientID, req.ActualAmount); err != nil {
		respondError(c, err)
		return
	}
	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "admin_set_actual_amount", "schedule_patient", req.SchedulePatientID, "")
	c.JSON(http.StatusOK, gin.H{"message": "Actual amount saved (admin)"})
}

// AdminListResults handles GET /api/admin/diagnosis-results
//
// Returns the paginated overview of all completed / no-show schedule patients
// sorted by schedule date + slot start time, most recent first.
func (h *DiagnosisHandler) AdminListResults(c *gin.Context) {
	var q dto.DiagnosisResultsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		respondBadRequest(c, err)
		return
	}
	resp, err := h.diagService.ListResults(c.Request.Context(), q)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// AdminExportResults handles GET /api/admin/export/diagnosis — downloads the
// diagnosis-results overview (per patient, with amounts) as xlsx, same filters
// as AdminListResults.
func (h *DiagnosisHandler) AdminExportResults(c *gin.Context) {
	var q dto.DiagnosisResultsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		respondBadRequest(c, err)
		return
	}
	f, err := h.diagService.BuildResultsExcel(c.Request.Context(), q)
	if err != nil {
		respondError(c, err)
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="diagnosis_results.xlsx"`)
	if err := f.Write(c.Writer); err != nil {
		respondCode(c, http.StatusInternalServerError, dto.CodeExportFailed, "Failed to generate Excel")
	}
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

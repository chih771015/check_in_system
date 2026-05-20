package handler

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"translator-checkin/internal/config"
	"translator-checkin/internal/dto"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// CheckinHandler handles check-in endpoints.
type CheckinHandler struct {
	checkinService *service.CheckinService
	exportService  *service.ExportService
	auditService   *service.AuditService
}

// NewCheckinHandler creates a new CheckinHandler.
func NewCheckinHandler(checkinService *service.CheckinService, exportService *service.ExportService, auditService *service.AuditService) *CheckinHandler {
	return &CheckinHandler{checkinService: checkinService, exportService: exportService, auditService: auditService}
}

// Checkin handles POST /api/checkins (multipart form with photos).
func (h *CheckinHandler) Checkin(c *gin.Context) {
	var req dto.CheckinRequest
	if err := c.ShouldBind(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		respondCode(c, http.StatusUnauthorized, dto.CodeUserContextMissing, "User not found in context")
		return
	}

	lat, _ := strconv.ParseFloat(c.PostForm("latitude"), 64)
	lng, _ := strconv.ParseFloat(c.PostForm("longitude"), 64)
	address := c.PostForm("address")

	selfieURL, err := saveUploadedFile(c, "selfie")
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeSelfieRequired, "Selfie photo is required: "+err.Error())
		return
	}

	// Stage 4: environment photo is optional. If supplied we still save it
	// so we keep parity with historical data; otherwise envURL stays empty.
	envURL := ""
	if u, errEnv := saveUploadedFile(c, "environment"); errEnv == nil {
		envURL = u
	}

	resp, err := h.checkinService.Checkin(
		c.Request.Context(),
		userID.(uint),
		req.ScheduleID,
		req.Type,
		lat, lng, address,
		selfieURL, envURL,
		false, "",
	)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// MakeupCheckin handles POST /api/checkins/makeup.
func (h *CheckinHandler) MakeupCheckin(c *gin.Context) {
	var req dto.CheckinMakeupRequest
	if err := c.ShouldBind(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		respondCode(c, http.StatusUnauthorized, dto.CodeUserContextMissing, "User not found in context")
		return
	}

	lat, _ := strconv.ParseFloat(c.PostForm("latitude"), 64)
	lng, _ := strconv.ParseFloat(c.PostForm("longitude"), 64)
	address := c.PostForm("address")

	selfieURL, err := saveUploadedFile(c, "selfie")
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeSelfieRequired, "Selfie photo is required: "+err.Error())
		return
	}

	// Stage 4: environment photo is optional. If supplied we still save it
	// so we keep parity with historical data; otherwise envURL stays empty.
	envURL := ""
	if u, errEnv := saveUploadedFile(c, "environment"); errEnv == nil {
		envURL = u
	}

	resp, err := h.checkinService.Checkin(
		c.Request.Context(),
		userID.(uint),
		req.ScheduleID,
		req.Type,
		lat, lng, address,
		selfieURL, envURL,
		true, req.MakeupReason,
	)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// AdminUpdateCheckin handles PUT /api/admin/checkins/:id.
func (h *CheckinHandler) AdminUpdateCheckin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidCheckinID, "Invalid checkin ID")
		return
	}

	var req dto.AdminUpdateCheckinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	ctx := c.Request.Context()
	if err := h.checkinService.AdminUpdateCheckin(ctx, uint(id), req); err != nil {
		respondError(c, err)
		return
	}
	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "update_checkin", "checkin", uint(id), "")
	c.JSON(http.StatusOK, gin.H{"message": "Checkin updated successfully"})
}

// AdminDeleteCheckin handles DELETE /api/admin/checkins/:id.
func (h *CheckinHandler) AdminDeleteCheckin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidCheckinID, "Invalid checkin ID")
		return
	}
	ctx := c.Request.Context()
	if err := h.checkinService.AdminDeleteCheckin(ctx, uint(id)); err != nil {
		respondError(c, err)
		return
	}
	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "delete_checkin", "checkin", uint(id), "")
	c.JSON(http.StatusOK, gin.H{"message": "Checkin deleted successfully"})
}

// MyCheckins handles GET /api/checkins for translators to view their own records.
func (h *CheckinHandler) MyCheckins(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		respondCode(c, http.StatusUnauthorized, dto.CodeUserContextMissing, "User not found in context")
		return
	}
	list, err := h.checkinService.MyHistory(c.Request.Context(), userID.(uint), c.Query("dateFrom"), c.Query("dateTo"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// MyStats handles GET /api/checkins/stats for translators to view their own stats.
func (h *CheckinHandler) MyStats(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		respondCode(c, http.StatusUnauthorized, dto.CodeUserContextMissing, "User not found in context")
		return
	}
	stats, err := h.checkinService.MyStats(c.Request.Context(), userID.(uint), c.Query("dateFrom"), c.Query("dateTo"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// AdminListCheckins handles GET /api/admin/checkins
func (h *CheckinHandler) AdminListCheckins(c *gin.Context) {
	params := service.AdminListParams{
		DateFrom:    c.Query("dateFrom"),
		DateTo:      c.Query("dateTo"),
		CheckinType: c.Query("type"),
	}

	if idStr := c.Query("translatorId"); idStr != "" {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err == nil {
			params.TranslatorID = uint(id)
		}
	}
	if isMakeupStr := c.Query("isMakeup"); isMakeupStr != "" {
		v := isMakeupStr == "true"
		params.IsMakeup = &v
	}

	checkins, err := h.checkinService.AdminList(c.Request.Context(), params)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": checkins})
}

// AdminExportExcel handles GET /api/admin/export/excel
func (h *CheckinHandler) AdminExportExcel(c *gin.Context) {
	params := parseExportParams(c)

	f, err := h.exportService.BuildCheckinExcel(c.Request.Context(), params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="checkins.xlsx"`)
	if err := f.Write(c.Writer); err != nil {
		respondCode(c, http.StatusInternalServerError, dto.CodeExportFailed, "Failed to generate Excel")
	}
}

// AdminExportGoogleSheet handles POST /api/admin/export/google-sheet
func (h *CheckinHandler) AdminExportGoogleSheet(c *gin.Context) {
	if config.AppConfig.GoogleCredentialsFile == "" {
		respondCode(c, http.StatusServiceUnavailable, dto.CodeGoogleNotConfigured, "Google credentials not configured. Set GOOGLE_CREDENTIALS_FILE env variable.")
		return
	}

	var req struct {
		Title string `json:"title"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Title == "" {
		req.Title = fmt.Sprintf("Checkin Records_%s", time.Now().Format("20060102_150405"))
	}

	params := parseExportParams(c)

	url, err := h.exportService.CreateCheckinGoogleSheet(c.Request.Context(), params, req.Title)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url, "title": req.Title})
}

func parseExportParams(c *gin.Context) service.AdminListParams {
	params := service.AdminListParams{
		DateFrom:    c.Query("dateFrom"),
		DateTo:      c.Query("dateTo"),
		CheckinType: c.Query("type"),
	}
	if idStr := c.Query("translatorId"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			params.TranslatorID = uint(id)
		}
	}
	return params
}

func saveUploadedFile(c *gin.Context, fieldName string) (string, error) {
	file, err := c.FormFile(fieldName)
	if err != nil {
		return "", fmt.Errorf("file field '%s' is required", fieldName)
	}
	return saveMultipartFile(c, file, fieldName)
}

// saveMultipartFile persists one *multipart.FileHeader to the upload dir and
// returns the URL path. Used for both single-photo checkin uploads and the
// stage-4 multi-photo diagnosis upload flow.
func saveMultipartFile(c *gin.Context, file *multipart.FileHeader, prefix string) (string, error) {
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s_%s_%d%s", prefix, time.Now().Format("20060102_150405"), time.Now().UnixNano(), ext)

	uploadDir := config.AppConfig.UploadDir
	savePath := filepath.Join(uploadDir, filename)

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return "/uploads/" + filename, nil
}

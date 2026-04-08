package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"translator-checkin/internal/config"
	"translator-checkin/internal/dto"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// CheckinHandler handles check-in endpoints.
type CheckinHandler struct {
	checkinService *service.CheckinService
}

// NewCheckinHandler creates a new CheckinHandler.
func NewCheckinHandler(checkinService *service.CheckinService) *CheckinHandler {
	return &CheckinHandler{checkinService: checkinService}
}

// Checkin handles POST /api/checkins (multipart form with photos).
func (h *CheckinHandler) Checkin(c *gin.Context) {
	var req dto.CheckinRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	// Parse GPS coordinates
	lat, _ := strconv.ParseFloat(c.PostForm("latitude"), 64)
	lng, _ := strconv.ParseFloat(c.PostForm("longitude"), 64)
	address := c.PostForm("address")

	// Save uploaded photos
	selfieURL, err := saveUploadedFile(c, "selfie")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Selfie photo is required: " + err.Error()})
		return
	}

	envURL, err := saveUploadedFile(c, "environment")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Environment photo is required: " + err.Error()})
		return
	}

	resp, err := h.checkinService.Checkin(
		userID.(uint),
		req.ScheduleID,
		req.Type,
		lat, lng, address,
		selfieURL, envURL,
		false, "",
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// MakeupCheckin handles POST /api/checkins/makeup (multipart form with photos + reason).
func (h *CheckinHandler) MakeupCheckin(c *gin.Context) {
	var req dto.CheckinMakeupRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	// Parse GPS coordinates
	lat, _ := strconv.ParseFloat(c.PostForm("latitude"), 64)
	lng, _ := strconv.ParseFloat(c.PostForm("longitude"), 64)
	address := c.PostForm("address")

	// Save uploaded photos
	selfieURL, err := saveUploadedFile(c, "selfie")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Selfie photo is required: " + err.Error()})
		return
	}

	envURL, err := saveUploadedFile(c, "environment")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Environment photo is required: " + err.Error()})
		return
	}

	resp, err := h.checkinService.Checkin(
		userID.(uint),
		req.ScheduleID,
		req.Type,
		lat, lng, address,
		selfieURL, envURL,
		true, req.MakeupReason,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
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

	checkins, err := h.checkinService.AdminList(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": checkins})
}

// AdminExportExcel handles GET /api/admin/export/excel
func (h *CheckinHandler) AdminExportExcel(c *gin.Context) {
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

	checkins, err := h.checkinService.AdminList(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	f := excelize.NewFile()
	sheet := "打卡紀錄"
	f.NewSheet(sheet)
	f.DeleteSheet("Sheet1")

	headers := []string{"打卡ID", "翻譯員ID", "翻譯員姓名", "打卡類型", "打卡時間", "地點", "GPS緯度", "GPS經度", "自拍照URL", "環境照URL", "是否補打卡", "補打卡原因"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	for rowIdx, ck := range checkins {
		row := rowIdx + 2
		typeLabel := "到達"
		if ck.Type == "leave" {
			typeLabel = "離開"
		}
		isMakeupLabel := "否"
		if ck.IsMakeup {
			isMakeupLabel = "是"
		}
		values := []interface{}{
			ck.ID,
			ck.TranslatorID,
			ck.TranslatorName,
			typeLabel,
			ck.CheckinTime.Format("2006-01-02 15:04:05"),
			ck.Address,
			ck.Latitude,
			ck.Longitude,
			ck.SelfieURL,
			ck.EnvironmentURL,
			isMakeupLabel,
			ck.MakeupReason,
		}
		for colIdx, val := range values {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			f.SetCellValue(sheet, cell, val)
		}
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="checkins.xlsx"`)
	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Excel"})
	}
}

// saveUploadedFile saves a multipart file and returns its URL path.
func saveUploadedFile(c *gin.Context, fieldName string) (string, error) {
	file, err := c.FormFile(fieldName)
	if err != nil {
		return "", fmt.Errorf("file field '%s' is required", fieldName)
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s_%s_%d%s", fieldName, time.Now().Format("20060102_150405"), time.Now().UnixNano(), ext)

	uploadDir := config.AppConfig.UploadDir
	savePath := filepath.Join(uploadDir, filename)

	// Ensure upload directory exists
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// Return URL path (relative to server root)
	return "/uploads/" + filename, nil
}

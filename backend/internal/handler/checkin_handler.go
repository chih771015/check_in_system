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

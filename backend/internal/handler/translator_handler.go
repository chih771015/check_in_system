package handler

import (
	"net/http"
	"strconv"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// TranslatorHandler handles admin endpoints for managing translators.
type TranslatorHandler struct {
	translatorService *service.TranslatorService
	authService       *service.AuthService
	auditService      *service.AuditService
}

// NewTranslatorHandler creates a new TranslatorHandler.
func NewTranslatorHandler(translatorService *service.TranslatorService, authService *service.AuthService, auditService *service.AuditService) *TranslatorHandler {
	return &TranslatorHandler{translatorService: translatorService, authService: authService, auditService: auditService}
}

// ListTranslators handles GET /api/admin/translators.
func (h *TranslatorHandler) ListTranslators(c *gin.Context) {
	status := c.Query("status")

	translators, err := h.translatorService.List(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": translators})
}

// CreateTranslator handles POST /api/admin/translators.
func (h *TranslatorHandler) CreateTranslator(c *gin.Context) {
	var req dto.CreateTranslatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := h.translatorService.Create(ctx, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "create_translator", "user", 0, "email="+req.Email+" name="+req.Name)

	c.JSON(http.StatusCreated, gin.H{"message": "Translator created successfully"})
}

// UpdateTranslator handles PUT /api/admin/translators/:id.
func (h *TranslatorHandler) UpdateTranslator(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid translator ID"})
		return
	}

	var req dto.UpdateTranslatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := h.translatorService.Update(ctx, uint(id), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "update_translator", "user", uint(id), "")

	c.JSON(http.StatusOK, gin.H{"message": "Translator updated successfully"})
}

// ResetTranslatorPassword handles POST /api/admin/translators/:id/reset-password.
// The admin supplies a new password; the target user is forced to change it on
// next login.
func (h *TranslatorHandler) ResetTranslatorPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid translator ID"})
		return
	}

	var req dto.AdminResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	adminID := c.GetUint("userID")
	if err := h.authService.AdminResetPassword(ctx, adminID, uint(id), req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.auditService.Log(ctx, adminID, "reset_password", "user", uint(id), "admin reset translator password")

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// DisableTranslator handles DELETE /api/admin/translators/:id.
func (h *TranslatorHandler) DisableTranslator(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid translator ID"})
		return
	}

	ctx := c.Request.Context()
	if err := h.translatorService.Disable(ctx, uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "disable_translator", "user", uint(id), "")

	c.JSON(http.StatusOK, gin.H{"message": "Translator disabled successfully"})
}

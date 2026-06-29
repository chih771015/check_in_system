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

	// page/pageSize are optional. Absent → pageSize 0 = return all (the dropdown
	// pickers rely on this); present → one page. The response always carries the
	// total so the management table can render a server-side pager.
	page, pageSize := 1, 0
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	translators, total, err := h.translatorService.List(c.Request.Context(), status, page, pageSize)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     translators,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CreateTranslator handles POST /api/admin/translators.
func (h *TranslatorHandler) CreateTranslator(c *gin.Context) {
	var req dto.CreateTranslatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	ctx := c.Request.Context()
	if err := h.translatorService.Create(ctx, req); err != nil {
		respondError(c, err)
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
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidTranslatorID, "Invalid translator ID")
		return
	}

	var req dto.UpdateTranslatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	ctx := c.Request.Context()
	if err := h.translatorService.Update(ctx, uint(id), req); err != nil {
		respondError(c, err)
		return
	}

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "update_translator", "user", uint(id), "")

	c.JSON(http.StatusOK, gin.H{"message": "Translator updated successfully"})
}

// ResetTranslatorPassword handles POST /api/admin/translators/:id/reset-password.
func (h *TranslatorHandler) ResetTranslatorPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidTranslatorID, "Invalid translator ID")
		return
	}

	var req dto.AdminResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	ctx := c.Request.Context()
	adminID := c.GetUint("userID")
	if err := h.authService.AdminResetPassword(ctx, adminID, uint(id), req.NewPassword); err != nil {
		respondError(c, err)
		return
	}

	h.auditService.Log(ctx, adminID, "reset_password", "user", uint(id), "admin reset translator password")

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// DisableTranslator handles DELETE /api/admin/translators/:id.
func (h *TranslatorHandler) DisableTranslator(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondCode(c, http.StatusBadRequest, dto.CodeInvalidTranslatorID, "Invalid translator ID")
		return
	}

	ctx := c.Request.Context()
	if err := h.translatorService.Disable(ctx, uint(id)); err != nil {
		respondError(c, err)
		return
	}

	adminID := c.GetUint("userID")
	h.auditService.Log(ctx, adminID, "disable_translator", "user", uint(id), "")

	c.JSON(http.StatusOK, gin.H{"message": "Translator disabled successfully"})
}

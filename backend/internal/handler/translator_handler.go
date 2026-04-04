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
}

// NewTranslatorHandler creates a new TranslatorHandler.
func NewTranslatorHandler(translatorService *service.TranslatorService) *TranslatorHandler {
	return &TranslatorHandler{translatorService: translatorService}
}

// ListTranslators handles GET /api/admin/translators.
func (h *TranslatorHandler) ListTranslators(c *gin.Context) {
	status := c.Query("status")

	translators, err := h.translatorService.List(status)
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

	if err := h.translatorService.Create(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	if err := h.translatorService.Update(uint(id), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Translator updated successfully"})
}

// DisableTranslator handles DELETE /api/admin/translators/:id.
func (h *TranslatorHandler) DisableTranslator(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid translator ID"})
		return
	}

	if err := h.translatorService.Disable(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Translator disabled successfully"})
}

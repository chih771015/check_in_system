package handler

import (
	"net/http"
	"strconv"

	"translator-checkin/internal/repository"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// AuditHandler exposes audit log endpoints to admins.
type AuditHandler struct {
	auditService *service.AuditService
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(auditService *service.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// ListAuditLogs handles GET /api/admin/audit-logs
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	f := repository.AuditLogFilter{
		Action:     c.Query("action"),
		TargetType: c.Query("targetType"),
		StartDate:  c.Query("startDate"),
		EndDate:    c.Query("endDate"),
	}
	if v := c.Query("adminId"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			f.AdminID = uint(id)
		}
	}
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			f.Page = p
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil {
			f.PageSize = ps
		}
	}

	logs, total, err := h.auditService.List(c.Request.Context(), f)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     logs,
		"total":    total,
		"page":     f.Page,
		"pageSize": f.PageSize,
	})
}

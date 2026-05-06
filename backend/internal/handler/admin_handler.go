package handler

import (
	"net/http"
	"strconv"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminHandler handles admin account management endpoints.
type AdminHandler struct {
	adminService *service.AdminService
	auditService *service.AuditService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(adminService *service.AdminService, auditService *service.AuditService) *AdminHandler {
	return &AdminHandler{adminService: adminService, auditService: auditService}
}

// ListAdmins handles GET /api/admin/admins
func (h *AdminHandler) ListAdmins(c *gin.Context) {
	admins, err := h.adminService.ListAdmins(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": admins})
}

// CreateAdmin handles POST /api/admin/admins
func (h *AdminHandler) CreateAdmin(c *gin.Context) {
	var req dto.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if err := h.adminService.CreateAdmin(ctx, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	requesterID := c.GetUint("userID")
	h.auditService.Log(ctx, requesterID, "create_admin", "user", 0, "email="+req.Email)
	c.JSON(http.StatusCreated, gin.H{"message": "管理員帳號建立成功"})
}

// DeleteAdmin handles DELETE /api/admin/admins/:id
func (h *AdminHandler) DeleteAdmin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的管理員 ID"})
		return
	}
	ctx := c.Request.Context()
	requesterID := c.GetUint("userID")
	if err := h.adminService.DeleteAdmin(ctx, requesterID, uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.auditService.Log(ctx, requesterID, "delete_admin", "user", uint(id), "")
	c.JSON(http.StatusOK, gin.H{"message": "管理員帳號已刪除"})
}

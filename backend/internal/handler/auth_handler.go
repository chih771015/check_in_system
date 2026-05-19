package handler

import (
	"net/http"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ChangePassword handles POST /api/auth/change-password.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err)
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		respondCode(c, http.StatusUnauthorized, dto.CodeUserContextMissing, "User not found in context")
		return
	}

	token, err := h.authService.ChangePassword(c.Request.Context(), userID.(uint), req.OldPassword, req.NewPassword)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ChangePasswordResponse{
		Message: "Password changed successfully",
		Token:   token,
	})
}

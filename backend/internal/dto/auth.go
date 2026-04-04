package dto

// LoginRequest represents a login request payload.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is returned on successful login.
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// ChangePasswordRequest is the payload for changing password.
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// UserResponse is the safe user representation sent to clients.
type UserResponse struct {
	ID           uint   `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	MustChangePW bool   `json:"mustChangePW"`
}

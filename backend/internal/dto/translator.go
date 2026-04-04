package dto

import "time"

// CreateTranslatorRequest is the payload for creating a new translator.
type CreateTranslatorRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone"`
}

// UpdateTranslatorRequest is the payload for updating a translator.
type UpdateTranslatorRequest struct {
	Name   *string `json:"name"`
	Phone  *string `json:"phone"`
	Status *string `json:"status"`
}

// TranslatorListResponse is a single item in the translators list.
type TranslatorListResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

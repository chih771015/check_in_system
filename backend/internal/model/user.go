package model

import "time"

// User represents the users table for admins and translators.
type User struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Email          string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash   string    `gorm:"type:varchar(255);not null" json:"-"`
	Name           string    `gorm:"type:varchar(255);not null" json:"name"`
	Phone          string    `gorm:"type:varchar(50)" json:"phone"`
	Role           string    `gorm:"type:varchar(20);not null" json:"role"`
	Status         string    `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	MustChangePW   bool      `gorm:"default:true" json:"must_change_pw"`
	LineUserID     string    `gorm:"type:varchar(255)" json:"line_user_id,omitempty"`
	TelegramChatID string    `gorm:"type:varchar(255)" json:"telegram_chat_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

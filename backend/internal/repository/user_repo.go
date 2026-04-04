package repository

import (
	"translator-checkin/internal/model"

	"gorm.io/gorm"
)

// UserRepository handles database operations for users.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByEmail returns a user by email address.
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID returns a user by ID.
func (r *UserRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindAll returns all translators, optionally filtered by status.
func (r *UserRepository) FindAll(status string) ([]model.User, error) {
	var users []model.User
	query := r.db.Where("role = ?", "translator")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Create inserts a new user record.
func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// Update saves changes to an existing user record.
func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

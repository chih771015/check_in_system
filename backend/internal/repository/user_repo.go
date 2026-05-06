package repository

import (
	"context"
	"time"

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

// WithCtx returns a shallow copy of the repository whose underlying *gorm.DB
// carries the request context. The GORM OTel plugin uses that context to
// nest SQL spans under the active HTTP span, so call this at the top of any
// request-scoped handler path to get a properly stitched trace.
func (r *UserRepository) WithCtx(ctx context.Context) *UserRepository {
	return &UserRepository{db: r.db.WithContext(ctx)}
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

// UpdatePasswordAndForceChange writes a new password hash and forces the user
// to change it on next login. Used by admin password reset.
func (r *UserRepository) UpdatePasswordAndForceChange(id uint, hash string) error {
	return r.db.Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash":  hash,
			"must_change_pw": true,
		}).Error
}

// IncrementLoginAttempts adds 1 to the login_attempts counter.
func (r *UserRepository) IncrementLoginAttempts(id uint) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("login_attempts", gorm.Expr("login_attempts + 1")).Error
}

// ResetLoginAttempts zeroes the login_attempts counter and clears locked_until.
func (r *UserRepository) ResetLoginAttempts(id uint) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Updates(map[string]any{"login_attempts": 0, "locked_until": nil}).Error
}

// LockUser sets locked_until to the given time.
func (r *UserRepository) LockUser(id uint, until time.Time) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("locked_until", until).Error
}

// FindAllAdmins returns all users with role = "admin", ordered by creation time.
func (r *UserRepository) FindAllAdmins() ([]model.User, error) {
	var users []model.User
	if err := r.db.Where("role = ?", "admin").
		Order("created_at ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// DeleteByID hard-deletes a user by primary key.
// The caller is responsible for verifying business rules (e.g. no self-delete).
func (r *UserRepository) DeleteByID(id uint) error {
	return r.db.Delete(&model.User{}, id).Error
}

package postgrestore

import (
	"time"

	"github.com/google/uuid"
)

type UserSchema struct {
	ID                uuid.UUID  `gorm:"column:id"`
	Email             string     `gorm:"column:email"`
	FirstName         string     `gorm:"column:first_name"`
	LastName          string     `gorm:"column:last_name"`
	AvatarURL         string     `gorm:"column:avatar_url"`
	IsActive          bool       `gorm:"column:is_active"`
	IsAdmin           bool       `gorm:"column:is_admin"`
	PasswordChangedAt *time.Time `gorm:"column:password_changed_at"`
	LastSignInAt      *time.Time `gorm:"column:last_signin_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
	DeletedAt         *time.Time `gorm:"column:deleted_at"`
}

func (UserSchema) TableName() string {
	return "users"
}

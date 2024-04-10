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

type FileSchema struct {
	ID        int        `gorm:"column:id"`
	Name      string     `gorm:"column:name"`
	Path      string     `gorm:"column:path"`
	FullPath  string     `gorm:"column:full_path"`
	Size      uint64     `gorm:"column:size"`
	Mode      uint32     `gorm:"column:mode"`
	MimeType  string     `gorm:"column:mime_type"`
	MD5       string     `gorm:"column:md5"`
	IsDir     bool       `gorm:"column:is_dir"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

func (FileSchema) TableName() string {
	return "files"
}

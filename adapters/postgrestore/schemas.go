package postgrestore

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
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
	RootID            uuid.UUID  `gorm:"column:root_id"`
	StorageUsage      uint64     `gorm:"column:storage_usage"`
	StorageCapacity   uint64     `gorm:"column:storage_capacity"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
	DeletedAt         *time.Time `gorm:"column:deleted_at"`
}

func (UserSchema) TableName() string {
	return "users"
}

func (s *UserSchema) ToDomainUser() *identity.User {
	return &identity.User{
		ID:                s.ID,
		Email:             s.Email,
		FirstName:         s.FirstName,
		LastName:          s.LastName,
		AvatarURL:         s.AvatarURL,
		IsActive:          s.IsActive,
		IsAdmin:           s.IsAdmin,
		PasswordChangedAt: s.PasswordChangedAt,
		LastSignInAt:      s.LastSignInAt,
		RootID:            s.RootID,
		StorageUsage:      s.StorageUsage,
		StorageCapacity:   s.StorageCapacity,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}

type FileSchema struct {
	ID            uuid.UUID  `gorm:"column:id"`
	Name          string     `gorm:"column:name"`
	Path          string     `gorm:"column:path"`
	PreviousPath  *string    `gorm:"column:previous_path"` // user for move to trash
	Size          uint64     `gorm:"column:size"`
	Mode          uint32     `gorm:"column:mode"`
	MimeType      string     `gorm:"column:mime_type"`
	MD5           string     `gorm:"column:md5"`
	IsDir         bool       `gorm:"column:is_dir"`
	GeneralAccess string     `gorm:"column:general_access"`
	OwnerID       uuid.UUID  `gorm:"column:owner_id"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at"`

	Owner *UserSchema `gorm:"foreignKey:OwnerID;references:ID"`
}

func (FileSchema) TableName() string {
	return "files"
}

func (f *FileSchema) FullPath() string {
	return filepath.Join(f.Path, f.Name)
}

func (s *FileSchema) ToDomainFile() *file.File {
	md5, _ := hex.DecodeString(s.MD5)

	var owner *identity.User
	if s.Owner != nil {
		owner = s.Owner.ToDomainUser()
	}

	return &file.File{
		ID:            s.ID,
		Name:          s.Name,
		Path:          s.Path,
		PreviousPath:  s.PreviousPath,
		Size:          s.Size,
		Mode:          os.FileMode(s.Mode),
		MimeType:      s.MimeType,
		MD5:           md5,
		IsDir:         s.IsDir,
		GeneralAccess: s.GeneralAccess,
		OwnerID:       s.OwnerID,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
		Owner:         owner,
	}
}

func (s *FileSchema) ToDomainSimpleFile() *file.SimpleFile {
	return &file.SimpleFile{
		ID:   s.ID,
		Name: s.Name,
		Path: s.Path,
	}
}

type ShareSchema struct {
	FileID    uuid.UUID `gorm:"column:file_id"`
	UserID    uuid.UUID `gorm:"column:user_id"`
	Role      string    `gorm:"column:role"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (ShareSchema) TableName() string {
	return "shares"
}

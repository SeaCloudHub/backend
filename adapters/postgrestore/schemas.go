package postgrestore

import (
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/google/uuid"
)

type UserSchema struct {
	ID                uuid.UUID      `gorm:"column:id"`
	Email             string         `gorm:"column:email"`
	FirstName         string         `gorm:"column:first_name"`
	LastName          string         `gorm:"column:last_name"`
	AvatarURL         string         `gorm:"column:avatar_url"`
	IsActive          bool           `gorm:"column:is_active"`
	IsAdmin           bool           `gorm:"column:is_admin"`
	PasswordChangedAt *time.Time     `gorm:"column:password_changed_at"`
	LastSignInAt      *time.Time     `gorm:"column:last_signin_at"`
	RootID            uuid.UUID      `gorm:"column:root_id"`
	StorageUsage      uint64         `gorm:"column:storage_usage"`
	StorageCapacity   uint64         `gorm:"column:storage_capacity"`
	CreatedAt         time.Time      `gorm:"column:created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at"`
	BlockedAt         *time.Time     `gorm:"column:blocked_at"`
}

func (UserSchema) TableName() string {
	return "users"
}

func (s *UserSchema) ToDomainUser() *identity.User {
	if s == nil {
		return nil
	}

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
		DeletedAt:         s.DeletedAt.Time,
		BlockedAt:         s.BlockedAt,
	}
}

type FileSchema struct {
	ID            uuid.UUID    `gorm:"column:id"`
	Name          string       `gorm:"column:name"`
	Path          string       `gorm:"column:path"`
	PreviousPath  *string      `gorm:"column:previous_path"` // user for move to trash
	Size          uint64       `gorm:"column:size"`
	Mode          uint32       `gorm:"column:mode"`
	MimeType      string       `gorm:"column:mime_type"`
	Type          string       `gorm:"column:type;->"`
	Thumbnail     *string      `gorm:"column:thumbnail"`
	MD5           string       `gorm:"column:md5"`
	IsDir         bool         `gorm:"column:is_dir"`
	GeneralAccess string       `gorm:"column:general_access"`
	OwnerID       uuid.UUID    `gorm:"column:owner_id"`
	CreatedAt     time.Time    `gorm:"column:created_at"`
	UpdatedAt     time.Time    `gorm:"column:updated_at"`
	DeletedAt     *time.Time   `gorm:"column:deleted_at"`
	FinishedAt    sql.NullTime `gorm:"column:finished_at"`

	Owner *UserSchema `gorm:"foreignKey:OwnerID;references:ID"`
}

func (FileSchema) TableName() string {
	return "files"
}

func (f *FileSchema) FullPath() string {
	return filepath.Join(f.Path, f.Name)
}

func (s *FileSchema) ToDomainFile() *file.File {
	if s == nil {
		return nil
	}

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
		Type:          s.Type,
		Thumbnail:     s.Thumbnail,
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

type StarSchema struct {
	FileID    uuid.UUID `gorm:"column:file_id"`
	UserID    uuid.UUID `gorm:"column:user_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (StarSchema) TableName() string { return "stars" }

type LogSchema struct {
	UserID    uuid.UUID `gorm:"column:user_id"`
	FileID    uuid.UUID `gorm:"column:file_id"`
	Action    string    `gorm:"column:action"`
	CreatedAt time.Time `gorm:"column:created_at"`

	File *FileSchema `gorm:"foreignKey:FileID;references:ID"`
	User *UserSchema `gorm:"foreignKey:UserID;references:ID"`
}

func (LogSchema) TableName() string { return "logs" }

func (s *LogSchema) ToDomainLog() *file.Log {
	if s == nil {
		return nil
	}

	return &file.Log{
		FileID:    s.FileID,
		UserID:    s.UserID,
		Action:    s.Action,
		CreatedAt: s.CreatedAt,
		File:      s.File.ToDomainFile(),
		User:      s.User.ToDomainUser(),
	}
}

func (s *LogSchema) ToDomainFile() *file.File {
	file := s.File.ToDomainFile()
	file.Log = s.ToDomainLog()
	file.Log.File = nil

	return file
}

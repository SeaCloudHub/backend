package identity

import (
	"context"
	"time"

	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/google/uuid"
)

type Store interface {
	Create(ctx context.Context, user *User) error
	UpdateAdmin(ctx context.Context, userID uuid.UUID) error
	UpdatePasswordChangedAt(ctx context.Context, userID uuid.UUID) error
	UpdateLastSignInAt(ctx context.Context, userID uuid.UUID) error
	UpdateRootID(ctx context.Context, userID, rootID uuid.UUID) error
	UpdateStorageUsage(ctx context.Context, userID uuid.UUID, usage uint64) error
	GetByID(ctx context.Context, userID string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetAll(ctx context.Context) ([]User, error)
	List(ctx context.Context, pagination *pagination.Pager, filter Filter) ([]User, error)
	ListByEmails(ctx context.Context, emails []string) ([]User, error)
	FuzzySearch(ctx context.Context, keyword string) ([]User, error)
	UpdateStorageCapacity(ctx context.Context, userID uuid.UUID, storageCapacity uint64) error
	ToggleActive(ctx context.Context, userID uuid.UUID) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, userID uuid.UUID) error
}

type User struct {
	ID                uuid.UUID  `json:"id"`
	Email             string     `json:"email"`
	FirstName         string     `json:"first_name"`
	LastName          string     `json:"last_name"`
	AvatarURL         string     `json:"avatar_url"`
	IsActive          bool       `json:"is_active"`
	IsAdmin           bool       `json:"is_admin"`
	PasswordChangedAt *time.Time `json:"password_changed_at"`
	LastSignInAt      *time.Time `json:"last_sign_in_at"`
	RootID            uuid.UUID  `json:"root_id"`
	StorageUsage      uint64     `json:"storage_usage"`
	StorageCapacity   uint64     `json:"storage_capacity"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         time.Time  `json:"deleted_at"`
	BlockedAt         *time.Time `json:"blocked_at"`
} // @name identity.User

func (u *User) WithName(firstName, lastName string) *User {
	u.FirstName = firstName
	u.LastName = lastName

	return u
}

func (u *User) WithAvatarURL(avatarURL string) *User {
	u.AvatarURL = avatarURL

	return u
}

func (u *User) UpdateInfo(firstName, lastName, avatarURL string) {
	u.FirstName = firstName
	u.LastName = lastName
	u.AvatarURL = avatarURL
}

type Filter struct {
	Keyword string
}

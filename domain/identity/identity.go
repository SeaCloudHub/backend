package identity

import (
	"context"
	"errors"
	"github.com/SeaCloudHub/backend/domain"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrIncorrectPassword  = errors.New("incorrect password")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidSession     = errors.New("invalid session")
	ErrSessionTooOld      = errors.New("session too old")
	ErrIdentityNotFound   = errors.New("identity not found")
)

type Service interface {
	Login(ctx context.Context, email string, password string) (*Session, error)
	WhoAmI(ctx context.Context, token string) (*Identity, error)
	ChangePassword(ctx context.Context, id *Identity, oldPassword string, newPassword string) error
	SetPasswordChangedAt(ctx context.Context, id *Identity) error
	GetByEmail(ctx context.Context, email string) (*Identity, error)

	// Admin APIs
	CreateIdentity(ctx context.Context, in SimpleIdentity, listener func(event domain.BaseDomainEvent) error) (*Identity, error)
	ListIdentities(ctx context.Context, pageToken string, pageSize int64) ([]ExtendedIdentity, string, error)
	CreateMultipleIdentities(ctx context.Context,
		simpleIdentities []SimpleIdentity, listener func(event domain.BaseDomainEvent) error) ([]*Identity, error)
}

type SimpleIdentity struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
	AvatarURL string
}

type Identity struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	Password          string     `json:"password,omitempty"`
	PasswordChangedAt *time.Time `json:"password_changed_at"`
	FirstName         string     `json:"first_name"`
	LastName          string     `json:"last_name"`
	AvatarURL         string     `json:"avatar_url"`
	Session           *Session   `json:"-"`
	IsAdmin           bool       `json:"is_admin"`
} // @name identity.Identity

type Session struct {
	ID        string     `json:"id"`
	Token     *string    `json:"token"`
	ExpiresAt *time.Time `json:"expires_at"`
	Identity  *Identity  `json:"identity"`
}

type ExtendedIdentity struct {
	Identity        `json:",inline"`
	LastAccessAt    *time.Time `json:"last_access_at"`
	UsedCapacity    uint64     `json:"used_capacity"`
	MaximumCapacity uint64     `json:"maximum_capacity"`
} // @name identity.ExtendedIdentity

func (i *Identity) WithLastAccessAt(t time.Time) *ExtendedIdentity {
	return &ExtendedIdentity{
		Identity:        *i,
		LastAccessAt:    &t,
		UsedCapacity:    0,
		MaximumCapacity: 0,
	}
}

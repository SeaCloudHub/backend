package identity

import (
	"context"
	"errors"
	"time"

	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrIncorrectPassword     = errors.New("incorrect password")
	ErrInvalidPassword       = errors.New("invalid password")
	ErrInvalidSession        = errors.New("invalid session")
	ErrSessionTooOld         = errors.New("session too old")
	ErrIdentityNotFound      = errors.New("identity not found")
	ErrIdentityWasDisabled   = errors.New("identity was disabled")
	ErrIdentityAlreadyExists = errors.New("identity already exists")
)

type Service interface {
	Login(ctx context.Context, email string, password string) (*Session, error)
	Logout(ctx context.Context, token string) error
	WhoAmI(ctx context.Context, token string) (*Identity, error)
	ChangePassword(ctx context.Context, id *Identity, oldPassword string, newPassword string) error
	GetByEmail(ctx context.Context, email string) (*Identity, error)
	GetByID(ctx context.Context, id string) (*Identity, error)

	// Admin APIs
	CreateIdentity(ctx context.Context, in SimpleIdentity) (*Identity, error)
	ListIdentities(ctx context.Context, paging *pagination.Cursor) ([]Identity, error)
	CreateMultipleIdentities(ctx context.Context, simpleIdentities []SimpleIdentity) ([]*Identity, error)
	UpdateIdentityState(ctx context.Context, id string, state string) error
	DeleteIdentity(ctx context.Context, id string) error
	ResetPassword(ctx context.Context, id *Identity, password string) error
}

type SimpleIdentity struct {
	Email    string
	Password string
}

type Identity struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	Password string   `json:"password,omitempty"`
	Session  *Session `json:"-"`
} // @name identity.Identity

func (id *Identity) ToUser() *User {
	return &User{
		ID:    uuid.MustParse(id.ID),
		Email: id.Email,
	}
}

type Session struct {
	ID        string     `json:"id"`
	Token     *string    `json:"token"`
	ExpiresAt *time.Time `json:"expires_at"`
	Identity  *Identity  `json:"identity"`
}

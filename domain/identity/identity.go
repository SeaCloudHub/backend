package identity

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidSession     = errors.New("invalid session")
)

type Service interface {
	Login(ctx context.Context, email string, password string) (*Session, error)
	WhoAmI(ctx context.Context, token string) (*Identity, error)
	ChangePassword(ctx context.Context, id *Identity, oldPassword string, newPassword string) error
	SyncPasswordChangedAt(ctx context.Context, id *Identity) error

	// Admin APIs
	CreateIdentity(ctx context.Context, email string, password string) (*Identity, error)
	ListIdentities(ctx context.Context, pageToken string, pageSize int64) ([]Identity, string, error)
}

type Identity struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	PasswordChangedAt *time.Time `json:"password_changed_at"`
	Session           *Session   `json:"-"`
}

type Session struct {
	ID        string     `json:"id"`
	Token     *string    `json:"token"`
	ExpiresAt *time.Time `json:"expires_at"`
}

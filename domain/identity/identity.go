package identity

import (
	"context"
	"errors"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidSession     = errors.New("invalid session")
)

type Service interface {
	Login(ctx context.Context, email string, password string) (string, error)
	WhoAmI(ctx context.Context, token string) (*Identity, error)
	CreateIdentity(ctx context.Context, email string, password string) (*Identity, error)
}

type Identity struct {
	ID string `json:"id"`
}

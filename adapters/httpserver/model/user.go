package model

import (
	"time"

	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/validation"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=32"`
} // @name model.LoginRequest

func (r *LoginRequest) Validate() error {
	return validation.Validate().Struct(r)
}

func (r *UpdateProfileRequest) Validate() error {
	return validation.Validate().Struct(r)
}

type LoginResponse struct {
	SessionID        string         `json:"session_id"`
	SessionToken     string         `json:"session_token"`
	SessionExpiresAt *time.Time     `json:"session_expires_at"`
	Identity         *identity.User `json:"identity"`
} // @name model.LoginResponse

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=6,max=32"`
	NewPassword string `json:"new_password" validate:"required,min=6,max=32"`
} // @name model.ChangePasswordRequest

type UpdateProfileRequest struct {
	Id        string `json:"id" validate:"required"`
	AvatarUrl string `json:"avatar_url"`
	FirstName string `json:"first_name" validate:"required,min=1"`
	LastName  string `json:"last_name" validate:"required,min=1"`
} // @name model.UpdateProfileRequest

type UpdateProfileResponse struct {
	Id string `json:"id"`
} // @name model.UpdateProfileResponse

func (r *ChangePasswordRequest) Validate() error {
	return validation.Validate().Struct(r)
}

type GetByEmailRequest struct {
	Email string `query:"email" validate:"required,email"`
} // @name model.GetByEmailRequest

func (r *GetByEmailRequest) Validate() error {
	return validation.Validate().Struct(r)
}

type GetByEmailResponse struct {
	Email             string     `json:"email"`
	FirstName         string     `json:"first_name"`
	LastName          string     `json:"last_name"`
	AvatarURL         string     `json:"avatar_url"`
	PasswordChangedAt *time.Time `json:"password_changed_at"`
} // @name model.GetByEmailResponse

type SuggestRequest struct {
	Query string `query:"query" validate:"required,min=2"`
} // @name model.SuggestRequest

func (r *SuggestRequest) Validate() error {
	return validation.Validate().Struct(r)
}

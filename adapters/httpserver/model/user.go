package model

import "github.com/SeaCloudHub/backend/pkg/validation"

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=32"`
} // @name model.LoginRequest

func (r *LoginRequest) Validate() error {
	return validation.Validate().Struct(r)
}

type LoginResponse struct {
	SessionToken string `json:"session_token"`
} // @name model.LoginResponse

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=6,max=32"`
	NewPassword string `json:"new_password" validate:"required,min=6,max=32"`
} // @name model.ChangePasswordRequest

func (r *ChangePasswordRequest) Validate() error {
	return validation.Validate().Struct(r)
}

package model

import "github.com/SeaCloudHub/backend/pkg/validation"

type AdminLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=32"`
}

func (r *AdminLoginRequest) Validate() error {
	return validation.Validate().Struct(r)
}

type AdminLoginResponse struct {
	SessionToken string `json:"session_token"`
}

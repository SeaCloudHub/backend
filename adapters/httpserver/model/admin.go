package model

import (
	"errors"

	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/validation"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type ListIdentitiesRequest struct {
	PageToken string `query:"page_token" validate:"omitempty"`
	PageSize  int64  `query:"page_size" validate:"omitempty,min=1,max=100"`
}

func (r *ListIdentitiesRequest) Validate() error {
	if r.PageSize == 0 {
		r.PageSize = 10
	}

	return validation.Validate().Struct(r)
}

type ListIdentitiesResponse struct {
	Identities []identity.Identity `json:"identities"`
	NextToken  string              `json:"next_token"`
}

type CreateIdentityRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

func (r *CreateIdentityRequest) Validate() error {
	if len(r.Password) == 0 {
		r.Password = gonanoid.Must(11)
	}

	return validation.Validate().Struct(r)
}

type ActivateAndDeactiveStateUserRequest struct {
	Id string `json:"id" validate:"required"`
}

type ActivateAndDeactiveStateUserResponse struct {
	Identitiy identity.Identity `json:"identity"`
}

func (r *ActivateAndDeactiveStateUserRequest) Validate() error {
	if r.Id == "" {
		return errors.New("id cannot be empty")
	}
	return validation.Validate().Struct(r)

}

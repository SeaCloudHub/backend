package model

import (
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
} // @name model.ListIdentitiesResponse

type CreateIdentityRequest struct {
	Email     string `json:"email" validate:"required,email" csv:"email"`
	Password  string `json:"password" validate:"required,min=8" csv:"password"`
	FirstName string `json:"first_name" validate:"omitempty,max=50" csv:"first_name"`
	LastName  string `json:"last_name" validate:"omitempty,max=50" csv:"last_name"`
	AvatarURL string `json:"avatar_url" validate:"omitempty,url" csv:"avatar_url"`
} // @name model.CreateIdentityRequest

func (r *CreateIdentityRequest) Validate() error {
	if len(r.Password) == 0 {
		r.Password = gonanoid.Must(11)
	}

	return validation.Validate().Struct(r)
}

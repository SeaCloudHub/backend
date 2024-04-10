package model

import (
	"context"

	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/SeaCloudHub/backend/pkg/validation"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type ListIdentitiesRequest struct {
	Limit int `query:"limit" validate:"required,min=1,max=100"`
	Page  int `query:"page" validate:"required,min=1"`
}

func (r *ListIdentitiesRequest) Validate() error {
	if r.Limit == 0 {
		r.Limit = 10
	}

	if r.Page == 0 {
		r.Page = 1
	}

	return validation.Validate().Struct(r)
}

type ListIdentitiesResponse struct {
	Identities []identity.ExtendedUser `json:"identities"`
	Pagination pagination.PageInfo     `json:"pagination"`
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

type UpdateIdentityStateRequest struct {
	ID    string `param:"identity_id" validate:"required,uuid" swaggerignore:"true"`
	State string `json:"state" validate:"required,oneof=active inactive"`
} // @name model.UpdateIdentityStateRequest

func (r *UpdateIdentityStateRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

package model

import (
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/validation"
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

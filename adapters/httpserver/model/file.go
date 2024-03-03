package model

import (
	"context"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/pkg/validation"
)

type ListEntriesRequest struct {
	DirPath string `query:"dirpath" validate:"required,dirpath"`
	Limit   int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor  string `query:"cursor" validate:"omitempty,base64"`
}

func (r *ListEntriesRequest) Validate(ctx context.Context) error {
	if r.Limit <= 0 {
		r.Limit = 10
	}

	return validation.Validate().StructCtx(ctx, r)
}

type ListEntriesResponse struct {
	Entries []file.Entry `json:"entries"`
	Cursor  string       `json:"cursor"`
}

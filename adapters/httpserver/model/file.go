package model

import (
	"context"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/SeaCloudHub/backend/pkg/validation"
)

type GetMetadataRequest struct {
	ID string `param:"id" validate:"required,uuid"`
}

func (r *GetMetadataRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type DownloadFileRequest struct {
	ID string `param:"id" validate:"required,uuid"`
}

func (r *DownloadFileRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type UploadFilesRequest struct {
	ID string `form:"id" validate:"required,uuid"`
}

func (r *UploadFilesRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type ListEntriesRequest struct {
	ID     string `param:"id" validate:"required,uuid" swaggerignore:"true"`
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string `query:"cursor" validate:"omitempty,base64url"`
}

func (r *ListEntriesRequest) Validate(ctx context.Context) error {
	if r.Limit <= 0 {
		r.Limit = 10
	}

	return validation.Validate().StructCtx(ctx, r)
}

type ListEntriesResponse struct {
	Entries []file.File `json:"entries"`
	Cursor  string      `json:"cursor"`
} // @name model.ListEntriesResponse

type ListPageEntriesRequest struct {
	ID    string `param:"id" validate:"required,uuid" swaggerignore:"true"`
	Page  int    `query:"page" validate:"required,min=1"`
	Limit int    `query:"limit" validate:"omitempty,min=1,max=100"`
} // @name model.ListPageEntriesRequest

func (r *ListPageEntriesRequest) Validate(ctx context.Context) error {
	if r.Limit == 0 {
		r.Limit = 10
	}

	if r.Page == 0 {
		r.Page = 1
	}

	return validation.Validate().StructCtx(ctx, r)
}

type ListPageEntriesResponse struct {
	Entries    []file.File         `json:"entries"`
	Pagination pagination.PageInfo `json:"pagination"`
} // @name model.ListPageEntriesResponse

type CreateDirectoryRequest struct {
	ID   string `json:"id" validate:"required,uuid"`
	Name string `json:"name" validate:"required,max=255"`
} // @name model.CreateDirectoryRequest

func (r *CreateDirectoryRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type ShareRequest struct {
	ID     string   `json:"id" validate:"required,uuid"`
	Emails []string `json:"emails" validate:"required,dive,email"`
	Role   string   `json:"role" validate:"required,oneof=viewer editor"`
} // @name model.ShareRequest

func (r *ShareRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type AccessRequest struct {
	ID string `param:"id" validate:"required,uuid"`
} // @name model.AccessRequest

func (r *AccessRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type UpdateGeneralAccessRequest struct {
	ID            string `json:"id" validate:"required,uuid"`
	GeneralAccess string `json:"general_access" validate:"required,oneof=restricted everyone-can-view everyone-can-edit"`
} // @name model.UpdateGeneralAccessRequest

func (r *UpdateGeneralAccessRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

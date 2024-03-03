package model

import (
	"context"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/pkg/validation"
)

type GetFileRequest struct {
	FilePath string `query:"file_path" validate:"required,filepath"`
}

func (r *GetFileRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type DownloadFileRequest struct {
	FilePath string `query:"file_path" validate:"required,filepath"`
}

func (r *DownloadFileRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type UploadFileResponse struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

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

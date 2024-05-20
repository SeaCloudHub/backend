package model

import (
	"context"
	"time"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/permission"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/SeaCloudHub/backend/pkg/validation"
)

type GetMetadataRequest struct {
	ID string `param:"id" validate:"required,uuid"`
}

func (r *GetMetadataRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type GetMetadataResponse struct {
	File    file.File             `json:"file"`
	Parents []file.SimpleFile     `json:"parents"`
	Users   []permission.FileUser `json:"users"`
} // @name model.GetMetadataResponse

type DownloadRequest struct {
	ID string `param:"id" validate:"required,uuid"`
} // @name model.DownloadRequest

func (r *DownloadRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type UploadFilesRequest struct {
	ID string `form:"id" validate:"required,uuid"`
}

func (r *UploadFilesRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type ListEntriesRequest struct {
	ID     string     `param:"id" validate:"required,uuid" swaggerignore:"true"`
	Limit  int        `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string     `query:"cursor" validate:"omitempty,base64url"`
	Type   string     `query:"type" validate:"omitempty,oneof=folder text document pdf json image video audio archive other"`
	After  *time.Time `query:"after" validate:"omitempty"`
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

type ListTrashRequest struct {
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string `query:"cursor" validate:"omitempty,base64url"`
}

func (r *ListTrashRequest) Validate(ctx context.Context) error {
	if r.Limit <= 0 {
		r.Limit = 10
	}

	return validation.Validate().StructCtx(ctx, r)
}

type ListTrashResponse struct {
	Entries []file.File `json:"entries"`
	Cursor  string      `json:"cursor"`
} // @name model.ListTrashResponse

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
	Name string `json:"name" validate:"required,max=255,ne=.trash"`
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

type Access struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	Role   string `json:"role" validate:"required,oneof=viewer editor revoked"`
} // @name model.AccessRequest

type UpdateAccessRequest struct {
	ID     string   `json:"id" validate:"required,uuid"`
	Access []Access `json:"access" validate:"required,dive"`
} // @name model.UpdateAccessRequest

func (r *UpdateAccessRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type CopyFilesRequest struct {
	IDs []string `json:"ids" validate:"required,dive,uuid"`
	To  string   `json:"to" validate:"required,uuid"`
} // @name model.CopyFilesRequest

func (r *CopyFilesRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type MoveRequest struct {
	ID        string   `json:"id" validate:"required,uuid"`
	SourceIDs []string `json:"source_ids" validate:"required,dive,uuid"`
	To        string   `json:"to" validate:"required,uuid"`
} // @name model.MoveRequest

func (r *MoveRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type MoveToTrashRequest struct {
	ID        string   `json:"id" validate:"required,uuid"`
	SourceIDs []string `json:"source_ids" validate:"required,dive,uuid"`
} // @name model.MoveToTrashRequest

func (r *MoveToTrashRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type RestoreFromTrashRequest struct {
	SourceIDs []string `json:"source_ids" validate:"required,dive,uuid"`
} // @name model.RestoreFromTrashRequest

func (r *RestoreFromTrashRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type RenameFileRequest struct {
	ID   string `json:"id" validate:"required,uuid"`
	Name string `json:"name" validate:"required,max=255"`
} // @name model.RenameFileRequest

func (r *RenameFileRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type DeleteRequest struct {
	SourceIDs []string `json:"source_ids" validate:"required,dive,uuid"`
} // @name model.DeleteRequest

func (r *DeleteRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type GetPermissionsRequest struct {
	ID string `param:"id" validate:"required,uuid"`
}

type SearchRequest struct {
	Query    string     `query:"query" validate:"required"`
	Limit    int        `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor   string     `query:"cursor" validate:"omitempty,base64url"`
	Type     string     `query:"type" validate:"omitempty,oneof=folder text document pdf json image video audio archive other"`
	After    *time.Time `query:"after" validate:"omitempty"`
	ParentID string     `query:"parent_id" validate:"omitempty,uuid"`
} // @name model.SearchRequest

func (r *SearchRequest) Validate(ctx context.Context) error {
	if r.Limit <= 0 {
		r.Limit = 10
	}

	return validation.Validate().StructCtx(ctx, r)
}

type SearchResponse struct {
	Entries []file.File `json:"entries"`
	Cursor  string      `json:"cursor"`
} // @name model.SearchResponse

type ListSuggestedRequest struct {
	Limit int  `query:"limit" validate:"omitempty,min=1,max=100"`
	Dir   bool `query:"dir" validate:"omitempty"`
} // @name model.ListSuggestedRequest

func (r *ListSuggestedRequest) Validate(ctx context.Context) error {
	if r.Limit <= 0 {
		r.Limit = 20
	}

	return validation.Validate().StructCtx(ctx, r)
}

type ListActivitiesRequest struct {
	ID     string `param:"id" validate:"required,uuid" swaggerignore:"true"`
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string `query:"cursor" validate:"omitempty,base64url"`
} // @name model.ListActivitiesRequest

func (r *ListActivitiesRequest) Validate(ctx context.Context) error {
	if r.Limit <= 0 {
		r.Limit = 100
	}

	return validation.Validate().StructCtx(ctx, r)
}

type ListActivitiesResponse struct {
	Activities []file.Log `json:"activities"`
	Cursor     string     `json:"cursor"`
} // @name model.ListActivitiesResponse

type GetStorageResponse struct {
	Capacity     uint64 `json:"capacity"`
	file.Storage `json:",inline"`
} // @name model.GetStorageResponse

type ListFileSizesRequest struct {
	Type   string     `query:"type" validate:"omitempty,oneof=text document pdf json image video audio archive other"`
	After  *time.Time `query:"after" validate:"omitempty"`
	Limit  int        `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string     `query:"cursor" validate:"omitempty,base64url"`
	Asc    bool       `query:"asc" validate:"omitempty"`
} // @name model.ListFileSizesRequest

func (r *ListFileSizesRequest) Validate(ctx context.Context) error {
	if r.Limit <= 0 {
		r.Limit = 30
	}

	return validation.Validate().StructCtx(ctx, r)
}

type ListFileSizesResponse struct {
	Entries []file.File `json:"entries"`
	Cursor  string      `json:"cursor"`
} // @name model.ListFileSizesResponse

type DownloadBatchRequest struct {
	ParentID string   `json:"parent_id" validate:"required,uuid"`
	IDs      []string `json:"ids" validate:"required,dive,uuid"`
} // @name model.DownloadBatchRequest

func (r *DownloadBatchRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type StarRequest struct {
	FileIDs []string `json:"file_ids" validate:"required,dive,uuid"`
} // @name model.StarRequest

func (r *StarRequest) Validate() error {
	return validation.Validate().Struct(r)
}

type UnstarRequest struct {
	FileIDs []string `json:"file_ids" validate:"required,dive,uuid"`
} // @name model.UnstarRequest

func (r *UnstarRequest) Validate() error {
	return validation.Validate().Struct(r)
}

type ListStarredRequest struct {
	Limit  int        `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string     `query:"cursor" validate:"omitempty,base64url"`
	Type   string     `query:"type" validate:"omitempty,oneof=folder text document pdf json image video audio archive other"`
	After  *time.Time `query:"after" validate:"omitempty"`
} // @name model.ListStarredRequest

func (r *ListStarredRequest) Validate(ctx context.Context) error {
	if r.Limit <= 0 {
		r.Limit = 10
	}

	return validation.Validate().StructCtx(ctx, r)
}

type ListStarredResponse struct {
	Entries []file.File `json:"entries"`
	Cursor  string      `json:"cursor"`
} // @name model.ListStarredResponse

type UploadChunkRequest struct {
	ID        string `form:"id" validate:"required,uuid"`       // Directory ID
	FileID    string `form:"file_id" validate:"omitempty,uuid"` // File ID
	TotalSize uint64 `form:"total_size" validate:"required_without=FileID"`
	Last      bool   `form:"last"`
} // @name model.UploadChunkRequest

func (r *UploadChunkRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

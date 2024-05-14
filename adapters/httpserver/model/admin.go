package model

import (
	"context"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/SeaCloudHub/backend/pkg/validation"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type ListIdentitiesRequest struct {
	Keyword string `query:"keyword" validate:"omitempty,max=50"`
	Limit   int    `query:"limit" validate:"required,min=1,max=100"`
	Page    int    `query:"page" validate:"required,min=1"`
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
	Identities []identity.User     `json:"identities"`
	Pagination pagination.PageInfo `json:"pagination"`
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
}

func (r *UpdateIdentityStateRequest) Validate(ctx context.Context) error {
	return validation.Validate().StructCtx(ctx, r)
}

type StatisticUser struct {
	TotalUsers   int `json:"total_users"`
	ActiveUsers  int `json:"active_users"`
	BlockedUsers int `json:"blocked_users"`
} // @name model.StatisticUser

type StatisticUserComparison struct {
	Name       string  `json:"name"`
	Value      int     `json:"value"`
	Percentage float64 `json:"percentage"`
} // @name model.StatisticUserComparison

func (s *StatisticUser) Compare(oldValue StatisticUser) []StatisticUserComparison {
	comparisons := []StatisticUserComparison{
		{
			Name:       "total_users",
			Value:      s.TotalUsers,
			Percentage: calculatePercentageChange(oldValue.TotalUsers, s.TotalUsers),
		},
		{
			Name:       "active_users",
			Value:      s.ActiveUsers,
			Percentage: calculatePercentageChange(oldValue.ActiveUsers, s.ActiveUsers),
		},
		{
			Name:       "blocked_users",
			Value:      s.BlockedUsers,
			Percentage: calculatePercentageChange(oldValue.BlockedUsers, s.BlockedUsers),
		},
	}

	return comparisons
}

func calculatePercentageChange(old, new int) float64 {
	if old == 0 {
		if new == 0 {
			return 0.0
		}
		return 100.0
	}
	return (float64(new-old) / float64(old)) * 100.0
}

type StatisticsResponse struct {
	StatisticUser        []StatisticUserComparison `json:"statistic_user"`
	StatisticUserByMonth map[string]StatisticUser  `json:"statistic_user_by_month"`
	TotalStorageUsage    uint64                    `json:"total_storage_usage"`
	TotalStorageCapacity uint64                    `json:"total_storage_capacity"`
	FileByType           map[string]uint           `json:"file_by_type"`
} // @name model.StatisticsResponse

type ChangeUserStorageCapacityRequest struct {
	StorageCapacity uint64 `json:"storage_capacity" validate:"required,min=0"`
} // @name model.ChangeUserStorageCapacityRequest

func (r *ChangeUserStorageCapacityRequest) Validate() error {
	return validation.Validate().Struct(r)
}

type GetUserFilesRequest struct {
	IdentityId string `param:"identity_id" validate:"required,uuid" swaggerignore:"true"`
	Page       int    `query:"page" validate:"required,min=1"`
	Limit      int    `query:"limit" validate:"omitempty,min=1,max=100"`
} // @name model.GetUserFilesRequest

func (r *GetUserFilesRequest) Validate(ctx context.Context) error {
	if r.Limit == 0 {
		r.Limit = 10
	}

	if r.Page == 0 {
		r.Page = 1
	}

	return validation.Validate().StructCtx(ctx, r)
}

type ListStoragesRequest struct {
	Limit int `query:"limit" validate:"required,min=1,max=100"`
	Page  int `query:"page" validate:"required,min=1"`
} // @name model.ListStoragesRequest

func (r *ListStoragesRequest) Validate() error {
	if r.Limit == 0 {
		r.Limit = 10
	}

	if r.Page == 0 {
		r.Page = 1
	}

	return validation.Validate().Struct(r)
}

type ListStoragesResponse struct {
	UserRootDirectories []file.File         `json:"user_root_directories"`
	Pagination          pagination.PageInfo `json:"pagination"`
} // @name model.ListStoragesResponse

type EditIdentityRequest struct {
	IdentityID string `param:"identity_id" validate:"required,uuid" swaggerignore:"true"`
	FirstName  string `json:"first_name" validate:"omitempty,max=50"`
	LastName   string `json:"last_name" validate:"omitempty,max=50"`
	AvatarURL  string `json:"avatar_url" validate:"omitempty,url"`
} // @name model.EditIdentityRequest

func (r *EditIdentityRequest) Validate() error {
	return validation.Validate().Struct(r)
}

type ResetPasswordResponse struct {
	Password string `json:"password"`
} // @name model.ResetPasswordResponse

type LogsRequest struct {
	UserID string `query:"user_id" validate:"omitempty,uuid"`
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string `query:"cursor" validate:"omitempty,base64url"`
} // @name model.LogsRequest

func (r *LogsRequest) Validate(ctx context.Context) error {
	if r.Limit == 0 {
		r.Limit = 20
	}

	return validation.Validate().Struct(r)
}

type LogsResponse struct {
	Logs   []file.Log `json:"logs"`
	Cursor string     `json:"cursor"`
} // @name model.LogsResponse

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

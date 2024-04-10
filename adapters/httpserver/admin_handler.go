package httpserver

import (
	"fmt"
	"maps"
	"net/http"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/SeaCloudHub/backend/pkg/mycontext"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/SeaCloudHub/backend/pkg/util"
	"github.com/labstack/echo/v4"
)

// AdminMe godoc
// @Summary AdminMe
// @Description AdminMe
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Success 200 {object} model.SuccessResponse{data=identity.User}
// @Failure 401 {object} model.ErrorResponse
// @Router /admin/me [get]
func (s *Server) AdminMe(c echo.Context) error {
	return s.success(c, c.Get(ContextKeyUser))
}

// ListIdentities godoc
// @Summary ListIdentities
// @Description ListIdentities
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param paging query model.ListIdentitiesRequest false "Paging"
// @Success 200 {object} model.SuccessResponse{data=model.ListIdentitiesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities [get]
func (s *Server) ListIdentities(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.ListIdentitiesRequest
	)

	// TODO: get maxCapacity from config of identity storage size
	// max capacity is 10GB for now
	const maxCapacity = 10 << 30

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	pager := pagination.NewPager(req.Page, req.Limit)

	users, err := s.UserStore.List(ctx, pager)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	extendedUsers := make([]identity.ExtendedUser, 0, len(users))

	for i, user := range users {
		extendedUsers = append(extendedUsers, user.Extend())

		fullPath := util.GetIdentityDirPath(user.ID.String())
		extendedUsers[i].StorageUsed, err = s.FileService.GetDirectorySize(ctx, fullPath)
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		extendedUsers[i].StorageCapacity = maxCapacity
	}

	return s.success(c, model.ListIdentitiesResponse{
		Identities: extendedUsers,
		Pagination: pager.PageInfo(),
	})
}

// CreateIdentity godoc
// @Summary CreateIdentity
// @Description CreateIdentity
// @Tags admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.CreateIdentityRequest true "Create identity request"
// @Success 200 {object} model.SuccessResponse{data=identity.Identity}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities [post]
func (s *Server) CreateIdentity(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.CreateIdentityRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, err := s.IdentityService.CreateIdentity(ctx, s.MapperService.ToIdentity(req))
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	id.Password = req.Password

	// create user
	user := id.ToUser().WithName(req.FirstName, req.LastName).WithAvatarURL(req.AvatarURL)

	if err := s.UserStore.Create(ctx, user); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// create user root directory
	fullPath := util.GetIdentityDirPath(id.ID)
	if err := s.FileService.CreateDirectory(ctx, fullPath); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// create user root directory permission
	if err := s.PermissionService.CreateDirectoryPermissions(ctx, id.ID, fullPath); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// get metadata
	entry, err := s.FileService.GetMetadata(ctx, fullPath)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// create files row
	f := entry.ToFile().WithPath("/")
	if err := s.FileStore.Create(ctx, f); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, id)
}

// CreateMultipleIdentities godoc
// @Summary CreateMultipleIdentities
// @Description CreateMultipleIdentities
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param file formData file true "CSV file"
// @Success 200 {object} model.SuccessResponse{data=[]identity.Identity}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities/bulk [post]
func (s *Server) CreateMultipleIdentities(c echo.Context) error {
	ctx := mycontext.NewEchoContextAdapter(c)

	file, _, err := c.Request().FormFile("file")
	if err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}
	defer file.Close()

	var req []model.CreateIdentityRequest

	err = s.CSVService.CsvToEntities(&file, &req)
	if err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	simpleIdentities, err := s.MapperService.ToIdentities(req)
	if err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	ids, err := s.IdentityService.CreateMultipleIdentities(ctx, simpleIdentities)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	for i := range ids {
		ids[i].Email = simpleIdentities[i].Email
		ids[i].Password = simpleIdentities[i].Password

		// create user
		user := ids[i].ToUser().WithName(req[i].FirstName, req[i].LastName).WithAvatarURL(req[i].AvatarURL)
		if err := s.UserStore.Create(ctx, user); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		// create user root directory
		fullPath := util.GetIdentityDirPath(ids[i].ID)
		if err := s.FileService.CreateDirectory(ctx, fullPath); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		// create user root directory permission
		if err := s.PermissionService.CreateDirectoryPermissions(ctx, ids[i].ID, fullPath); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		// get metadata
		entry, err := s.FileService.GetMetadata(ctx, fullPath)
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		// create files row
		f := entry.ToFile().WithPath("/")
		if err := s.FileStore.Create(ctx, f); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	return s.success(c, ids)
}

// UpdateIdentityState godoc
// @Summary UpdateIdentityState
// @Description UpdateIdentityState
// @Tags admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param identity_id path string true "Identity ID"
// @Param payload body model.UpdateIdentityStateRequest true "Update identity state request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities/{identity_id}/state [patch]
func (s *Server) UpdateIdentityState(c echo.Context) error {
	ctx := mycontext.NewEchoContextAdapter(c)

	var req model.UpdateIdentityStateRequest
	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	if err := s.IdentityService.UpdateIdentityState(ctx, req.ID, req.State); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, nil)
}

// DownloadIdentitiesTemplate godoc
// @Summary Download Identities Template CSV
// @Description Download a CSV template file for creating identities.
// @Tags admin
// @Produce text/csv
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Success 200 {file} file "CSV file"
// @Failure 401 {object} model.ErrorResponse
// @Router /admin/identities/template [get]
func (s *Server) DownloadIdentitiesTemplate(c echo.Context) error {
	templateData := []model.CreateIdentityRequest{{}}

	buf, err := s.CSVService.EntitiesToCsv(templateData)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Set the headers
	c.Response().Header().Set(echo.HeaderContentType, "text/csv")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", "identities.csv"))

	return c.Blob(http.StatusOK, "text/csv", buf)
}

// Dashboard godoc
// @Summary Dashboard
// @Description Dashboard
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Success 200 {object} model.SuccessResponse{data=map[string]interface{}}
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/dashboard [get]
func (s *Server) Dashboard(c echo.Context) error {
	var ctx = mycontext.NewEchoContextAdapter(c)

	dirStatus, err := s.FileService.DirStatus(ctx)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	volStatus, err := s.FileService.VolStatus(ctx)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	maps.Copy(dirStatus, volStatus)

	return s.success(c, dirStatus)
}

func (s *Server) RegisterAdminRoutes(router *echo.Group) {
	router.Use(s.adminMiddleware)
	router.GET("/me", s.AdminMe)
	router.GET("/dashboard", s.Dashboard)

	router.Use(s.passwordChangedAtMiddleware)
	router.GET("/identities", s.ListIdentities)
	router.POST("/identities", s.CreateIdentity)
	router.POST("/identities/bulk", s.CreateMultipleIdentities)
	router.GET("/identities/template", s.DownloadIdentitiesTemplate)
	router.PATCH("/identities/:identity_id/state", s.UpdateIdentityState)
}

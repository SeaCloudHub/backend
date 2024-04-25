package httpserver

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"path/filepath"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/app"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/google/uuid"
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
		ctx = app.NewEchoContextAdapter(c)
		req model.ListIdentitiesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	pager := pagination.NewPager(req.Page, req.Limit)
	filter := identity.Filter{Keyword: req.Keyword}

	users, err := s.UserStore.List(ctx, pager, filter)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, model.ListIdentitiesResponse{
		Identities: users,
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
		ctx = app.NewEchoContextAdapter(c)
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

	user := id.ToUser().WithName(req.FirstName, req.LastName).WithAvatarURL(req.AvatarURL)

	// get root directory
	rootDir, err := s.FileStore.GetByFullPath(ctx, "/")
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.createUser(ctx, user, rootDir.ID.String()); err != nil {
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
	ctx := app.NewEchoContextAdapter(c)

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

	// get root directory
	rootDir, err := s.FileStore.GetByFullPath(ctx, "/")
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	for i := range ids {
		ids[i].Email = simpleIdentities[i].Email
		ids[i].Password = simpleIdentities[i].Password

		user := ids[i].ToUser().WithName(req[i].FirstName, req[i].LastName).WithAvatarURL(req[i].AvatarURL)

		if err := s.createUser(ctx, user, rootDir.ID.String()); err != nil {
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
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities/{identity_id}/state [patch]
func (s *Server) UpdateIdentityState(c echo.Context) error {
	ctx := app.NewEchoContextAdapter(c)

	user, err := s.UserStore.GetByID(ctx, uuid.MustParse(c.Param(
		"identity_id")))
	if err != nil {
		if errors.Is(err, identity.ErrIdentityNotFound) {
			return s.error(c, apperror.ErrIdentityNotFound(err))
		}
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if user.IsAdmin {
		return s.error(c, apperror.ErrForbidden(errors.New("cannot update admin user")))
	}

	var req model.UpdateIdentityStateRequest
	req.ID = user.ID.String()
	if user.IsActive {
		req.State = "inactive"
	} else {
		req.State = "active"
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	if err := s.UserStore.ToggleActive(ctx, user.ID); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
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
	var ctx = app.NewEchoContextAdapter(c)

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

// Statistics godoc
// @Summary Statistics
// @Description Statistics
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Success 200 {object} model.SuccessResponse{data=model.StatisticsResponse}
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/statistics [get]
func (s *Server) Statistics(c echo.Context) error {
	var ctx = app.NewEchoContextAdapter(c)

	users, err := s.UserStore.GetAll(ctx)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	totalUsers := len(users)
	activeUsers := 0
	var totalStorageUsage uint64
	for _, user := range users {
		if user.IsActive {
			activeUsers++
		}

		totalStorageUsage += user.StorageUsage
	}
	blockedUsers := totalUsers - activeUsers

	resp := model.StatisticsResponse{
		TotalUsers:        totalUsers,
		ActiveUsers:       activeUsers,
		BlockedUsers:      blockedUsers,
		TotalStorageUsage: totalStorageUsage,
	}

	return s.success(c, resp)
}

func (s *Server) RegisterAdminRoutes(router *echo.Group) {
	router.Use(s.adminMiddleware)
	router.GET("/me", s.AdminMe)
	router.GET("/dashboard", s.Dashboard)
	router.GET("/statistics", s.Statistics)

	router.Use(s.passwordChangedAtMiddleware)
	router.GET("/identities", s.ListIdentities)
	router.POST("/identities", s.CreateIdentity)
	router.POST("/identities/bulk", s.CreateMultipleIdentities)
	router.GET("/identities/template", s.DownloadIdentitiesTemplate)
	router.PATCH("/identities/:identity_id/state", s.UpdateIdentityState)
}

func (s *Server) createUser(ctx context.Context, user *identity.User, rootID string) error {
	userID := user.ID.String()

	// create user
	if err := s.UserStore.Create(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	// create user root directory
	fullPath := app.GetIdentityDirPath(userID)
	if err := s.FileService.CreateDirectory(ctx, fullPath); err != nil {
		return fmt.Errorf("create user root directory: %w", err)
	}

	// get metadata
	entry, err := s.FileService.GetMetadata(ctx, fullPath)
	if err != nil {
		return fmt.Errorf("get metadata: %w", err)
	}

	// create files row
	f := entry.ToFile().WithID(uuid.New()).WithPath("/").WithOwnerID(user.ID)
	if err := s.FileStore.Create(ctx, f); err != nil {
		return fmt.Errorf("create files row: %w", err)
	}

	// create user root directory permission
	if err := s.PermissionService.CreateDirectoryPermissions(ctx, userID, f.ID.String(), rootID); err != nil {
		return fmt.Errorf("create user root directory permissions: %w", err)
	}

	// update user root id
	if err := s.UserStore.UpdateRootID(ctx, user.ID, f.ID); err != nil {
		return fmt.Errorf("update user root id: %w", err)
	}

	// create trash directory
	trashPath := filepath.Join(fullPath, ".trash") + string(filepath.Separator)
	if err := s.FileService.CreateDirectory(ctx, trashPath); err != nil {
		return fmt.Errorf("create user trash directory: %w", err)
	}

	// get metadata
	trashEntry, err := s.FileService.GetMetadata(ctx, trashPath)
	if err != nil {
		return fmt.Errorf("get metadata: %w", err)
	}

	// create files row
	trash := trashEntry.ToFile().WithID(uuid.New()).WithPath(fullPath).WithOwnerID(user.ID)
	if err := s.FileStore.Create(ctx, trash); err != nil {
		return fmt.Errorf("create files row: %w", err)
	}

	// create trash directory permissions
	if err := s.PermissionService.CreateDirectoryPermissions(ctx, userID, trash.ID.String(), f.ID.String()); err != nil {
		return fmt.Errorf("create user trash directory permissions: %w", err)
	}

	return nil
}

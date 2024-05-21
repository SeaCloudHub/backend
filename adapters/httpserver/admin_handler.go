package httpserver

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/file"
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
	rootDir, err := s.FileStore.GetRootDirectory(ctx)
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
	rootDir, err := s.FileStore.GetRootDirectory(ctx)
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

	user, err := s.UserStore.GetByID(ctx, c.Param("identity_id"))
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

	var totalStorageUsage uint64

	statisticUserByMonthMap := make(map[string]model.StatisticUser)
	for _, user := range users {
		month := ""
		if user.IsActive {
			month = user.CreatedAt.Format("2006-01")
		} else {
			month = user.BlockedAt.Format("2006-01")
		}

		statisticUser := statisticUserByMonthMap[month]

		// Update statistics based on user's status
		if user.IsActive {
			statisticUser.ActiveUsers++
		} else {
			statisticUser.BlockedUsers++
		}
		statisticUser.TotalUsers++
		statisticUserByMonthMap[month] = statisticUser

		// Update totalStorageUsage
		totalStorageUsage += user.StorageUsage
	}

	currentMonth := time.Now().Format("2006-01")
	statisticUser, ok := statisticUserByMonthMap[currentMonth]
	if !ok {
		statisticUser = model.StatisticUser{}
	}

	lasMonth := time.Now().AddDate(0, -1, 0).Format("2006-01")
	lastStatisticUser, ok := statisticUserByMonthMap[lasMonth]
	if !ok {
		lastStatisticUser = model.StatisticUser{}
	}

	comparison := statisticUser.Compare(lastStatisticUser)

	files, err := s.FileStore.GetAllFiles(ctx)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}
	fileByType := make(map[string]uint)
	for _, f := range files {
		fileByType[f.Type]++
	}

	resp := model.StatisticsResponse{
		StatisticUser:        comparison,
		StatisticUserByMonth: statisticUserByMonthMap,
		TotalStorageUsage:    totalStorageUsage,
		TotalStorageCapacity: 30 << 30, // 30GB
		FileByType:           fileByType,
	}

	return s.success(c, resp)
}

// ChangeUserStorageCapacity godoc
// @Summary Change user's storage capacity
// @Description Change user's storage capacity
// @Tags admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.ChangeUserStorageCapacityRequest true "Change user's storage capacity request"
// @Param identity_id path string true "Identity ID"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities/{identity_id}/storage [patch]
func (s *Server) ChangeUserStorageCapacity(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ChangeUserStorageCapacityRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, err := s.UserStore.GetByID(ctx, c.Param("identity_id"))
	if err != nil {
		if errors.Is(err, identity.ErrIdentityNotFound) {
			return s.error(c, apperror.ErrIdentityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if req.StorageCapacity < user.StorageUsage {
		return s.error(c, apperror.ErrInvalidParam(errors.New("storage capacity must be greater than storage usage")))
	}

	if err := s.UserStore.UpdateStorageCapacity(ctx, user.ID, req.StorageCapacity); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, nil)
}

// GetIdentityDetails godoc
// @Summary Get user details
// @Description Get user details
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param identity_id path string true "Identity ID"
// @Success 200 {object} model.SuccessResponse{data=identity.User}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities/{identity_id} [get]
func (s *Server) GetIdentityDetails(c echo.Context) error {
	var ctx = app.NewEchoContextAdapter(c)

	user, err := s.UserStore.GetByID(ctx, c.Param("identity_id"))
	if err != nil {
		if errors.Is(err, identity.ErrIdentityNotFound) {
			return s.error(c, apperror.ErrIdentityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, user)
}

// GetIdentityFiles godoc
// @Summary Get user files
// @Description Get user files
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param identity_id path string true "Identity ID"
// @Param request query model.GetUserFilesRequest true "Get user files request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities/{identity_id}/files [get]
func (s *Server) GetIdentityFiles(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.GetUserFilesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, err := s.UserStore.GetByID(ctx, req.IdentityId)
	if err != nil {
		if errors.Is(err, identity.ErrIdentityNotFound) {
			return s.error(c, apperror.ErrIdentityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	userDir, err := s.FileStore.GetByID(ctx, user.RootID.String())
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	pager := pagination.NewPager(req.Page, req.Limit)
	files, err := s.FileStore.ListPager(ctx, userDir.FullPath(), pager)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	for i := range files {
		files[i] = *files[i].Response()
	}

	return s.success(c, model.ListPageEntriesResponse{
		Entries:    files,
		Pagination: pager.PageInfo(),
	})
}

// ListStorages godoc
// @Summary ListStorages
// @Description ListStorages
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param paging query model.ListStoragesRequest false "Paging"
// @Success 200 {object} model.SuccessResponse{data=model.ListStoragesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/storages [get]
func (s *Server) ListStorages(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ListStoragesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	pager := pagination.NewPager(req.Page, req.Limit)
	rootDirectories, err := s.FileStore.ListRootDirectory(ctx, pager)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	for i := range rootDirectories {
		rootDirectories[i] = *rootDirectories[i].Response()
	}

	return s.success(c, model.ListStoragesResponse{
		UserRootDirectories: rootDirectories,
		Pagination:          pager.PageInfo(),
	})
}

// EditIdentity godoc
// @Summary EditIdentity
// @Description EditIdentity
// @Tags admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param identity_id path string true "Identity ID"
// @Param payload body model.EditIdentityRequest true "Edit identity request"
// @Success 200 {object} model.SuccessResponse{data=identity.User}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities/{identity_id} [patch]
func (s *Server) EditIdentity(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.EditIdentityRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, err := s.UserStore.GetByID(ctx, req.IdentityID)
	if err != nil {
		if errors.Is(err, identity.ErrIdentityNotFound) {
			return s.error(c, apperror.ErrIdentityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	user.UpdateInfo(req.FirstName, req.LastName, req.AvatarURL)
	if err := s.UserStore.Update(ctx, user); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, user)
}

// DeleteIdentity godoc
// @Summary DeleteIdentity
// @Description DeleteIdentity
// @Tags admin
// @Produce json
// @Param Authorization header string true " Bearer token" default(Bearer <session_token>)
// @Param identity_id path string true "Identity ID"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities/{identity_id} [delete]
func (s *Server) DeleteIdentity(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
	)

	user, err := s.UserStore.GetByID(ctx, c.Param("identity_id"))
	if err != nil {
		if errors.Is(err, identity.ErrIdentityNotFound) {
			return s.error(c, apperror.ErrIdentityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if user.IsAdmin {
		return s.error(c, apperror.ErrForbidden(errors.New("cannot delete admin user")))
	}

	files, err := s.FileStore.ListUserFiles(ctx, user.ID)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Delete user
	if err := s.UserStore.Delete(ctx, user.ID); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Delete identity
	if err := s.IdentityService.DeleteIdentity(ctx, user.ID.String()); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Delete user files
	if err := s.FileStore.DeleteUserFiles(ctx, user.ID); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Delete shared files by user
	if err := s.FileStore.DeleteShareByUserID(ctx, user.ID); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Delete starred files by user
	if err := s.FileStore.DeleteStarByUserID(ctx, user.ID); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Delete permissions
	if err := s.PermissionService.DeleteUserPermissions(ctx, user.ID.String()); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	for _, f := range files {
		// Delete file from storage
		if err := s.FileService.Delete(ctx, f.ID.String()); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		// Delete file permissions
		if f.IsDir {
			if err := s.PermissionService.DeleteDirectoryPermissions(ctx, f.ID.String()); err != nil {
				return s.error(c, apperror.ErrInternalServer(err))
			}
		} else {
			if err := s.PermissionService.DeleteFilePermissions(ctx, f.ID.String()); err != nil {
				return s.error(c, apperror.ErrInternalServer(err))
			}
		}

		// Delete shared files
		if err := s.FileStore.DeleteShareByFileID(ctx, f.ID); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		// Delete starred files
		if err := s.FileStore.DeleteStarByFileID(ctx, f.ID); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	return s.success(c, nil)
}

// ResetPassword godoc
// @Summary ResetPassword
// @Description ResetPassword
// @Tags admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param identity_id path string true "Identity ID"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities/{identity_id}/reset-password [patch]
func (s *Server) ResetPassword(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
	)

	identityID := uuid.MustParse(c.Param("identity_id"))

	e, err := s.IdentityService.GetByID(ctx, identityID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	password := gonanoid.Must(11)
	if err := s.IdentityService.ResetPassword(ctx, e, password); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, model.ResetPasswordResponse{Password: password})
}

// Logs godoc
// @Summary Logs
// @Description Logs
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request query model.LogsRequest false "Request"
// @Success 200 {object} model.SuccessResponse{data=model.LogsResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/logs [get]
func (s *Server) Logs(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.LogsRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	cursor := pagination.NewCursor(req.Cursor, req.Limit)
	logs, err := s.FileStore.ReadLogs(ctx, req.UserID, cursor)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, model.LogsResponse{
		Logs:   logs,
		Cursor: cursor.NextToken(),
	})
}

func (s *Server) RegisterAdminRoutes(router *echo.Group) {
	router.Use(s.adminMiddleware)
	router.GET("/me", s.AdminMe)
	router.GET("/dashboard", s.Dashboard)
	router.GET("/statistics", s.Statistics)
	router.GET("/logs", s.Logs)

	router.Use(s.passwordChangedAtMiddleware)
	router.GET("/identities", s.ListIdentities)
	router.GET("/identities/:identity_id", s.GetIdentityDetails)
	router.PATCH("/identities/:identity_id", s.EditIdentity)
	router.DELETE("/identities/:identity_id", s.DeleteIdentity)
	router.PATCH("/identities/:identity_id/reset-password", s.ResetPassword)
	router.POST("/identities", s.CreateIdentity)
	router.POST("/identities/bulk", s.CreateMultipleIdentities)
	router.GET("/identities/template", s.DownloadIdentitiesTemplate)
	router.PATCH("/identities/:identity_id/state", s.UpdateIdentityState)
	router.PATCH("/identities/:identity_id/storage", s.ChangeUserStorageCapacity)
	router.GET("/identities/:identity_id/files", s.GetIdentityFiles)

	router.GET("/storages", s.ListStorages)
}

func (s *Server) createUser(ctx context.Context, user *identity.User, rootID string) error {
	userID := user.ID.String()

	// create user
	if err := s.UserStore.Create(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	// create user root directory
	fullPath := app.GetIdentityDirPath(userID)

	// create files row
	f := file.NewDirectory(userID).WithID(uuid.New()).WithPath("/").WithOwnerID(user.ID)
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

	// create files row
	trash := file.NewDirectory(".trash").WithID(uuid.New()).WithPath(fullPath).WithOwnerID(user.ID)
	if err := s.FileStore.Create(ctx, trash); err != nil {
		return fmt.Errorf("create files row: %w", err)
	}

	// create trash directory permissions
	if err := s.PermissionService.CreateDirectoryPermissions(ctx, userID, trash.ID.String(), f.ID.String()); err != nil {
		return fmt.Errorf("create user trash directory permissions: %w", err)
	}

	return nil
}

package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/SeaCloudHub/backend/pkg/app"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/gammazero/workerpool"
	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/domain/permission"

	"github.com/labstack/echo/v4"
)

// GetMetadata godoc
// @Summary GetMetadata
// @Description GetMetadata
// @Tags file
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request path model.GetMetadataRequest true "Get metadata request"
// @Success 200 {object} model.SuccessResponse{data=file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id}/metadata [get]
func (s *Server) GetMetadata(c echo.Context) error {
	var (
		ctx     = app.NewEchoContextAdapter(c)
		req     model.GetMetadataRequest
		canView bool
		err     error
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	f, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !f.IsDir {
		canView, err = s.PermissionService.CanViewFile(ctx, id.ID, f.ID.String())
	} else {
		canView, err = s.PermissionService.CanViewDirectory(ctx, id.ID, f.ID.String())
	}

	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
	}

	return s.success(c, f.Response())
}

// Download godoc
// @Summary Download
// @Description Download
// @Tags file
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request path model.DownloadFileRequest true "Download file request"
// @Success 200 {file} file
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id}/download [get]
func (s *Server) Download(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.DownloadFileRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if e.IsDir {
		return s.error(c, apperror.ErrFileOnlyOperation())
	}

	canView, err := s.PermissionService.CanViewFile(ctx, id.ID, e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
	}

	f, mime, err := s.FileService.DownloadFile(ctx, e.FullPath)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}
	defer f.Close()

	return c.Stream(http.StatusOK, mime, f)
}

// UploadFiles godoc
// @Summary UploadFiles
// @Description UploadFiles
// @Tags file
// @Accept multipart/form-data
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request formData model.UploadFilesRequest true "Upload files request"
// @Param files formData file true "Files"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files [post]
func (s *Server) UploadFiles(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.UploadFilesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Check if user has permission to upload files
	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	// Files
	form, err := c.MultipartForm()
	if err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	files := form.File["files"]

	totalSize := lo.Reduce(files, func(agg uint64, file *multipart.FileHeader, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	if totalSize+e.Owner.StorageUsage > e.Owner.StorageCapacity {
		return s.error(c, apperror.ErrStorageCapacityExceeded())
	}

	wp := workerpool.New(10)
	var m sync.Mutex
	var resp []file.File

	for _, file := range files {
		// open file
		src, err := file.Open()
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
		defer src.Close()

		fullPath := filepath.Join(e.FullPath, file.Filename)

		// TODO: handle file already exists

		wp.Submit(func() {
			// save files
			f, err := s.createFile(ctx, e, src, fullPath, user.ID)
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *f.Response())
		})
	}

	wp.StopWait()

	newStorageUsage := lo.Reduce(resp, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, e.Owner.StorageUsage)

	// update user storage usage
	if err := s.UserStore.UpdateStorageUsage(ctx, e.OwnerID, newStorageUsage); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, resp)
}

// ListEntries godoc
// @Summary ListEntries
// @Description ListEntries
// @Tags file
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param id path string true "Directory ID"
// @Param request query model.ListEntriesRequest true "List entries request"
// @Success 200 {object} model.SuccessResponse{data=model.ListEntriesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id} [get]
func (s *Server) ListEntries(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ListEntriesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !e.IsDir {
		return s.error(c, apperror.ErrDirectoryOnlyOperation())
	}

	identity, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	canView, err := s.PermissionService.CanViewDirectory(ctx, identity.ID, e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
	}

	cursor := pagination.NewCursor(req.Cursor, req.Limit)
	files, err := s.FileStore.ListCursor(ctx, e.FullPath, cursor)
	if err != nil {
		if errors.Is(err, file.ErrInvalidCursor) {
			return s.error(c, apperror.ErrInvalidParam(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	for i := range files {
		files[i] = *files[i].Response()
	}

	return s.success(c, model.ListEntriesResponse{
		Entries: files,
		Cursor:  cursor.NextToken(),
	})
}

// ListPageEntries godoc
// @Summary ListPageEntries
// @Description ListPageEntries
// @Tags file
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param id path string true "Directory ID"
// @Param request query model.ListPageEntriesRequest true "List page entries request"
// @Success 200 {object} model.SuccessResponse{data=model.ListPageEntriesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id}/page [get]
func (s *Server) ListPageEntries(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ListPageEntriesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	identity, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !e.IsDir {
		return s.error(c, apperror.ErrDirectoryOnlyOperation())
	}

	canView, err := s.PermissionService.CanViewDirectory(ctx, identity.ID, e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
	}

	pager := pagination.NewPager(req.Page, req.Limit)
	files, err := s.FileStore.ListPager(ctx, e.FullPath, pager)
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

// CreateDirectory godoc
// @Summary CreateDirectory
// @Description CreateDirectory
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.CreateDirectoryRequest true "Create directory request"
// @Success 200 {object} model.SuccessResponse{data=file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/directories [post]
func (s *Server) CreateDirectory(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.CreateDirectoryRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	parent, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), parent.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	dirpath := filepath.Join(parent.FullPath, req.Name) + string(filepath.Separator)
	if err := s.FileService.CreateDirectory(ctx, dirpath); err != nil {
		if errors.Is(err, file.ErrDirAlreadyExists) {
			return s.error(c, apperror.ErrDirAlreadyExists(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	entry, err := s.FileService.GetMetadata(ctx, dirpath)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	f := entry.ToFile().WithID(uuid.New()).WithPath(parent.FullPath).WithOwnerID(user.ID)
	if err := s.FileStore.Create(ctx, f); err != nil {
		if errors.Is(err, file.ErrDirAlreadyExists) {
			return s.error(c, apperror.ErrDirAlreadyExists(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.PermissionService.CreateDirectoryPermissions(ctx, user.ID.String(), f.ID.String(), parent.ID.String()); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, f.Response())
}

// Share godoc
// @Summary Share
// @Description Share
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.ShareRequest true "Share request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/share [post]
func (s *Server) Share(c echo.Context) error {
	var (
		ctx     = app.NewEchoContextAdapter(c)
		req     model.ShareRequest
		canEdit bool
		err     error
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if e.IsDir {
		canEdit, err = s.PermissionService.CanEditDirectory(ctx, user.ID.String(), e.ID.String())
	} else {
		canEdit, err = s.PermissionService.CanEditFile(ctx, user.ID.String(), e.ID.String())
	}

	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	users, err := s.UserStore.ListByEmails(ctx, req.Emails)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	userIDs := lo.Map(users, func(u identity.User, index int) uuid.UUID {
		return u.ID
	})

	if err := s.FileStore.UpsertShare(ctx, e.ID, userIDs, req.Role); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// TODO: notify users

	return s.success(c, nil)
}

// Access godoc
// @Summary Access
// @Description Access
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request path model.AccessRequest true "Access request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id}/access [get]
func (s *Server) Access(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.AccessRequest
		err error
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	var role string

	switch e.GeneralAccess {
	case "everyone-can-view":
		role = "viewer"
	case "everyone-can-edit":
		role = "editor"

	case "restricted":
		fallthrough
	default:
		share, err := s.FileStore.GetShare(ctx, e.ID, user.ID)
		if err != nil {
			if errors.Is(err, file.ErrNotFound) {
				return s.success(c, nil)
			}

			return s.error(c, apperror.ErrInternalServer(err))
		}

		role = share.Role
	}

	// clear permissions
	if e.IsDir {
		if err := s.PermissionService.ClearDirectoryPermissions(ctx, e.ID.String(), user.ID.String()); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	} else {
		if err := s.PermissionService.ClearFilePermissions(ctx, e.ID.String(), user.ID.String()); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	// add permissions
	if err := s.PermissionService.CreatePermission(ctx, permission.NewCreatePermission(
		user.ID.String(), e.ID.String(), e.IsDir, role)); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// remove share
	if err := s.FileStore.DeleteShare(ctx, e.ID, user.ID); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, nil)
}

// UpdateGeneralAccess godoc
// @Summary UpdateGeneralAccess
// @Description UpdateGeneralAccess
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.UpdateGeneralAccessRequest true "Update general access request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/general-access [patch]
func (s *Server) UpdateGeneralAccess(c echo.Context) error {
	var (
		ctx     = app.NewEchoContextAdapter(c)
		req     model.UpdateGeneralAccessRequest
		canEdit bool
		err     error
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if e.IsDir {
		canEdit, err = s.PermissionService.CanEditDirectory(ctx, user.ID.String(), e.ID.String())
	} else {
		canEdit, err = s.PermissionService.CanEditFile(ctx, user.ID.String(), e.ID.String())
	}

	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	if err := s.FileStore.UpdateGeneralAccess(ctx, e.ID, req.GeneralAccess); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, nil)
}

// CopyFiles godoc
// @Summary CopyFiles
// @Description CopyFiles
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.CopyFilesRequest true "Copy files request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/copy [post]
func (s *Server) CopyFiles(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.CopyFilesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	// check if user has edit permission to the destination directory
	dest, err := s.FileStore.GetByID(ctx, req.To)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), dest.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	files, err := s.FileStore.ListByIDs(ctx, req.IDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	totalSize := lo.Reduce(files, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	if totalSize+dest.Owner.StorageUsage > dest.Owner.StorageCapacity {
		return s.error(c, apperror.ErrStorageCapacityExceeded())
	}

	var resp []file.File

	wp := workerpool.New(10)
	var m sync.Mutex

	for _, e := range files {
		// copy directory is not allowed
		if e.IsDir {
			return s.error(c, apperror.ErrFileOnlyOperation())
		}

		// check if user has view permission to the file
		canView, err := s.PermissionService.CanViewFile(ctx, user.ID.String(), e.ID.String())
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		if !canView {
			return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
		}

		// copy file
		wp.Submit(func() {
			src, _, err := s.FileService.DownloadFile(ctx, e.FullPath)
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}
			defer src.Close()

			newName := fmt.Sprintf("Copy #%s of %s", gonanoid.MustGenerate("0123456789ABCDEF", 3), e.Name)
			fullPath := filepath.Join(dest.FullPath, newName)

			f, err := s.createFile(ctx, dest, src, fullPath, user.ID)
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *f.Response())
		})
	}

	wp.StopWait()

	newStorageUsage := lo.Reduce(resp, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, dest.Owner.StorageUsage)

	// update user storage usage
	if err := s.UserStore.UpdateStorageUsage(ctx, dest.OwnerID, newStorageUsage); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, resp)

}

// Move godoc
// @Summary Move
// @Description Move
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.MoveRequest true "Move files request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/move [post]
func (s *Server) Move(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.MoveRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	// check if user has edit permission to the source directory
	src, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), src.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	// check if user has edit permission to the destination directory
	dest, err := s.FileStore.GetByID(ctx, req.To)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err = s.PermissionService.CanEditDirectory(ctx, user.ID.String(), dest.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	files, err := s.FileStore.ListSelectedChildren(ctx, src, req.SourceIDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	totalSize := lo.Reduce(files, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	if totalSize+dest.Owner.StorageUsage > dest.Owner.StorageCapacity {
		return s.error(c, apperror.ErrStorageCapacityExceeded())
	}

	var resp []file.File

	wp := workerpool.New(10)
	var m sync.Mutex

	for _, e := range files {
		wp.Submit(func() {
			dstFullPath := strings.Replace(e.FullPath, src.FullPath, dest.FullPath, 1)
			dstPath := strings.Replace(e.Path, src.FullPath, dest.FullPath, 1)

			if e.Path == src.FullPath {
				// move top level files
				if err := s.FileService.Move(ctx, e.FullPath, dstPath); err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}

				// update parent relationship
				if e.IsDir {
					err = s.PermissionService.UpdateDirectoryParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
				} else {
					err = s.PermissionService.UpdateFileParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
				}

				if err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}
			}

			f := e.WithPath(dstPath).WithFullPath(dstFullPath)
			if err := s.FileStore.UpdatePath(ctx, e.ID, dstPath, dstFullPath); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *f.Response())
		})
	}

	wp.StopWait()

	if dest.OwnerID == src.OwnerID {
		return s.success(c, resp)
	}

	totalSize = lo.Reduce(resp, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	// update user storage usage
	if err := s.UserStore.UpdateStorageUsage(ctx, dest.OwnerID, dest.Owner.StorageUsage+totalSize); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.UserStore.UpdateStorageUsage(ctx, src.OwnerID, src.Owner.StorageUsage-totalSize); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, resp)
}

// Rename godoc
// @Summary Rename
// @Description Rename
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.RenameFileRequest true "Rename file request"
// @Success 200 {object} model.SuccessResponse{data=file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/rename [patch]
func (s *Server) Rename(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.RenameFileRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := func() (bool, error) {
		if e.IsDir {
			return s.PermissionService.CanEditDirectory(ctx, user.ID.String(), e.ID.String())
		}
		return s.PermissionService.CanEditFile(ctx, user.ID.String(), e.ID.String())
	}()
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	newFullPath := strings.Replace(e.FullPath, e.Name, req.Name, 1)
	newPath := strings.Replace(e.Path, e.Name, req.Name, 1)

	if err := s.FileService.Rename(ctx, strings.TrimRight(e.FullPath, string(filepath.Separator)), strings.TrimRight(newFullPath,
		string(filepath.Separator))); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.FileStore.UpdateName(ctx, e.ID, req.Name); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	resp := *e.WithName(req.Name).WithPath(newPath).WithFullPath(
		newFullPath).Response()

	return s.success(c, resp)
}

// MoveToTrash godoc
// @Summary MoveToTrash
// @Description MoveToTrash
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.MoveToTrashRequest true "Move to trash request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/move/trash [post]
func (s *Server) MoveToTrash(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.MoveToTrashRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	// check if user has edit permission to the source directory
	src, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), src.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	// get trash directory
	dest, err := s.FileStore.GetTrashByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err = s.PermissionService.CanEditDirectory(ctx, user.ID.String(), dest.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	files, err := s.FileStore.ListSelectedOwnedChildren(ctx, user.ID, src, req.SourceIDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	totalSize := lo.Reduce(files, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	if totalSize+dest.Owner.StorageUsage > dest.Owner.StorageCapacity {
		return s.error(c, apperror.ErrStorageCapacityExceeded())
	}

	var resp []file.File

	wp := workerpool.New(10)
	var m sync.Mutex

	for _, e := range files {
		wp.Submit(func() {
			dstFullPath := strings.Replace(e.FullPath, src.FullPath, dest.FullPath, 1)
			dstPath := strings.Replace(e.Path, src.FullPath, dest.FullPath, 1)

			if e.Path == src.FullPath {
				// move top level files
				if err := s.FileService.Move(ctx, e.FullPath, dstPath); err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}

				// update parent relationship
				if e.IsDir {
					err = s.PermissionService.UpdateDirectoryParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
				} else {
					err = s.PermissionService.UpdateFileParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
				}

				if err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}
			}

			f := e.WithPath(dstPath).WithFullPath(dstFullPath)
			if err := s.FileStore.MoveToTrash(ctx, e.ID, dstPath, dstFullPath); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *f.Response())
		})
	}

	wp.StopWait()

	if dest.OwnerID == src.OwnerID {
		return s.success(c, resp)
	}

	totalSize = lo.Reduce(resp, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	// update user storage usage
	if err := s.UserStore.UpdateStorageUsage(ctx, dest.OwnerID, dest.Owner.StorageUsage+totalSize); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.UserStore.UpdateStorageUsage(ctx, src.OwnerID, src.Owner.StorageUsage-totalSize); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, resp)
}

// RestoreFromTrash godoc
// @Summary RestoreFromTrash
// @Description RestoreFromTrash
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.RestoreFromTrashRequest true "Restore from trash request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/restore [post]
func (s *Server) RestoreFromTrash(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.RestoreFromTrashRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	// check if user has edit permission to the trash directory
	src, err := s.FileStore.GetTrashByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), src.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	files, err := s.FileStore.ListSelected(ctx, src, req.SourceIDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	var resp []file.File

	wp := workerpool.New(10)
	var m sync.Mutex

	for _, e := range files {
		if e.PreviousPath == nil || *e.PreviousPath == "" {
			continue
		}

		wp.Submit(func() {
			dest, err := s.FileStore.GetByFullPath(ctx, *e.PreviousPath)
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			// check if user has edit permission to the destination directory
			canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), dest.ID.String())
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			if !canEdit {
				// if user has no edit permission to the destination directory, restore to root directory
				dest, err = s.FileStore.GetByID(ctx, e.Owner.RootID.String())
				if err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}
			}

			dstFullPath := strings.Replace(e.FullPath, src.FullPath, dest.FullPath, 1)
			dstPath := strings.Replace(e.Path, src.FullPath, dest.FullPath, 1)
			path := e.Path

			if err := s.FileService.Move(ctx, e.FullPath, dstPath); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			// update parent relationship
			if e.IsDir {
				err = s.PermissionService.UpdateDirectoryParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
			} else {
				err = s.PermissionService.UpdateFileParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
			}

			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			f := e.WithPath(dstPath).WithFullPath(dstFullPath)
			if err := s.FileStore.RestoreFromTrash(ctx, e.ID, dstPath, dstFullPath); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			totalSize := e.Size

			if e.IsDir {
				children, err := s.FileStore.RestoreChildrenFromTrash(ctx, path, dstPath)
				if err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}

				totalSize = lo.Reduce(children, func(agg uint64, file file.File, index int) uint64 {
					return agg + uint64(file.Size)
				}, totalSize)
			}

			if dest.OwnerID == src.OwnerID {
				return
			}

			// update user storage usage
			if err := s.UserStore.UpdateStorageUsage(ctx, dest.OwnerID, dest.Owner.StorageUsage+totalSize); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			if err := s.UserStore.UpdateStorageUsage(ctx, src.OwnerID, src.Owner.StorageUsage-totalSize); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *f.Response())
		})
	}

	wp.StopWait()

	return s.success(c, resp)
}

// Delete godoc
// @Summary Delete
// @Description Delete
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.DeleteRequest true "Delete files request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/delete [post]
func (s *Server) Delete(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.DeleteRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	files, err := s.FileStore.ListByIDs(ctx, req.SourceIDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	var (
		resp      []file.File
		totalSize uint64
	)

	wp := workerpool.New(10)
	var m sync.Mutex

	for _, e := range files {
		wp.Submit(func() {
			// check if user can delete the file
			var (
				canDelete bool
				err       error
			)

			if e.IsDir {
				canDelete, err = s.PermissionService.CanDeleteDirectory(ctx, user.ID.String(), e.ID.String())
			} else {
				canDelete, err = s.PermissionService.CanDeleteFile(ctx, user.ID.String(), e.ID.String())
			}

			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			if !canDelete {
				s.Logger.Errorw(permission.ErrNotPermittedToDelete.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			if err := s.FileService.Delete(ctx, e.FullPath); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			files, err := s.FileStore.Delete(ctx, e)
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			// delete file permissions
			if e.IsDir {
				if err := s.PermissionService.DeleteDirectoryPermissions(ctx, e.ID.String()); err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}

				for _, f := range files {
					totalSize += uint64(f.Size)

					if f.IsDir {
						if err := s.PermissionService.DeleteDirectoryPermissions(ctx, f.ID.String()); err != nil {
							s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
							return
						}

						return
					}

					if err := s.PermissionService.DeleteFilePermissions(ctx, f.ID.String()); err != nil {
						s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
						return
					}
				}
			} else {
				totalSize = uint64(e.Size)

				if err := s.PermissionService.DeleteFilePermissions(ctx, e.ID.String()); err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *e.Response())
		})
	}

	// update user storage usage
	if totalSize > 0 {
		if err := s.UserStore.UpdateStorageUsage(ctx, files[0].Owner.ID, files[0].Owner.StorageUsage-totalSize); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	wp.StopWait()

	return s.success(c, resp)
}

func (s *Server) RegisterFileRoutes(router *echo.Group) {
	router.Use(s.passwordChangedAtMiddleware)
	router.POST("/directories", s.CreateDirectory)
	router.POST("/share", s.Share) // share file or directory with some users
	router.POST("/copy", s.CopyFiles)
	router.POST("/move", s.Move)
	router.PATCH("/rename", s.Rename)
	router.POST("/move/trash", s.MoveToTrash)
	router.POST("/restore", s.RestoreFromTrash)
	router.POST("/delete", s.Delete)
	router.POST("", s.UploadFiles)
	router.PATCH("/general-access", s.UpdateGeneralAccess)
	router.GET("/:id", s.ListEntries)
	router.GET("/:id/page", s.ListPageEntries)
	router.GET("/:id/metadata", s.GetMetadata)
	router.GET("/:id/download", s.Download)
	router.GET("/:id/access", s.Access) // get access to the shared file or directory
}

func (s *Server) createFile(ctx context.Context, parent *file.File, reader io.Reader, fullPath string, ownerID uuid.UUID) (*file.File, error) {
	_, err := s.FileService.CreateFile(ctx, reader, fullPath)
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}

	entry, err := s.FileService.GetMetadata(ctx, fullPath)
	if err != nil {
		return nil, fmt.Errorf("get metadata: %w", err)
	}

	f := entry.ToFile().WithID(uuid.New()).WithPath(parent.FullPath).WithOwnerID(ownerID)
	if err := s.FileStore.Create(ctx, f); err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	// create file permissions
	if err := s.PermissionService.CreateFilePermissions(ctx, ownerID.String(), f.ID.String(), parent.ID.String()); err != nil {
		return nil, fmt.Errorf("create file permissions: %w", err)
	}

	return f, nil
}

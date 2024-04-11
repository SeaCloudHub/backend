package httpserver

import (
	"errors"
	"net/http"
	"path/filepath"

	"github.com/SeaCloudHub/backend/pkg/app"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/google/uuid"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"

	"github.com/labstack/echo/v4"
)

// GetMetadata godoc
// @Summary GetMetadata
// @Description GetMetadata
// @Tags file
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param full_path query string true "File or directory full path"
// @Success 200 {object} model.SuccessResponse{data=file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/metadata [get]
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
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	} else {
		canView, err = s.PermissionService.CanViewDirectory(ctx, id.ID, f.ID.String())
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(errors.New("not permitted to view")))
	}

	return s.success(c, f.Response())
}

// Download godoc
// @Summary Download
// @Description Download
// @Tags file
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param file_path query string true "File path"
// @Success 200 {file} file
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/download [get]
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
		return s.error(c, apperror.ErrInvalidParam(errors.New("cannot download directory")))
	}

	canView, err := s.PermissionService.CanViewFile(ctx, id.ID, e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(errors.New("not permitted to view")))
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
// @Param dirpath formData string true "Directory path"
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
		return s.error(c, apperror.ErrForbidden(errors.New("not permitted to edit")))
	}

	// Files
	form, err := c.MultipartForm()
	if err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	var resp []file.File

	// TODO: add workerpool to handle multiple file uploads concurrently
	files := form.File["files"]
	for _, file := range files {
		// open file
		src, err := file.Open()
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
		defer src.Close()

		fullPath := filepath.Join(e.FullPath, file.Filename)

		// TODO: handle file already exists

		// save files
		_, err = s.FileService.CreateFile(ctx, src, fullPath)
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		entry, err := s.FileService.GetMetadata(ctx, fullPath)
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		f := entry.ToFile().WithID(uuid.New()).WithPath(e.FullPath).WithOwnerID(user.ID)
		if err := s.FileStore.Create(ctx, f); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		// create file permissions
		if err := s.PermissionService.CreateFilePermissions(ctx, user.ID.String(), f.ID.String(), e.ID.String()); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		resp = append(resp, *f.Response())
	}

	return s.success(c, resp)
}

// ListEntries godoc
// @Summary ListEntries
// @Description ListEntries
// @Tags file
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param dirpath query string true "Directory path"
// @Param limit query int false "Limit"
// @Param cursor query string false "Cursor"
// @Success 200 {object} model.SuccessResponse{data=model.ListEntriesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files [get]
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
		return s.error(c, apperror.ErrEntityNotFound(errors.New("not a directory")))
	}

	identity, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	canView, err := s.PermissionService.CanViewDirectory(ctx, identity.ID, e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(errors.New("not permitted to view")))
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
// @Param dirpath query string true "Directory path"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} model.SuccessResponse{data=model.ListPageEntriesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/page [get]
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
		return s.error(c, apperror.ErrEntityNotFound(errors.New("not a directory")))
	}

	canView, err := s.PermissionService.CanViewDirectory(ctx, identity.ID, e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(errors.New("not permitted to view")))
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
		return s.error(c, apperror.ErrForbidden(errors.New("not permitted to edit")))
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

func (s *Server) RegisterFileRoutes(router *echo.Group) {
	router.Use(s.passwordChangedAtMiddleware)
	router.POST("/directories", s.CreateDirectory)
	router.POST("", s.UploadFiles)
	router.GET("/:id", s.ListEntries)
	router.GET("/:id/page", s.ListPageEntries)
	router.GET("/:id/metadata", s.GetMetadata)
	router.GET("/:id/download", s.Download)
}

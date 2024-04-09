package httpserver

import (
	"errors"
	"net/http"
	"path/filepath"

	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/SeaCloudHub/backend/pkg/pagination"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/mycontext"
	"github.com/SeaCloudHub/backend/pkg/validation"

	"github.com/labstack/echo/v4"
)

// GetMetadata godoc
// @Summary GetMetadata
// @Description GetMetadata
// @Tags file
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param path query string true "File or directory path"
// @Success 200 {object} model.SuccessResponse{data=file.Entry}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/metadata [get]
func (s *Server) GetMetadata(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.GetMetadata
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	f, err := s.FileService.GetMetadata(ctx, filepath.Join(id.ID, req.Path))
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, f)
}

// DownloadFile godoc
// @Summary DownloadFile
// @Description DownloadFile
// @Tags file
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param file_path query string true "File path"
// @Success 200 {file} file
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/download [get]
func (s *Server) DownloadFile(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.DownloadFileRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	f, mime, err := s.FileService.DownloadFile(ctx, filepath.Join(id.ID, req.FilePath))
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
// @Success 200 {object} model.SuccessResponse{data=[]model.UploadFileResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files [post]
func (s *Server) UploadFiles(c echo.Context) error {
	var ctx = mycontext.NewEchoContextAdapter(c)

	// Identity ID will be used as root directory
	identity, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	// Directory
	dirpath := c.FormValue("dirpath")
	if err := validation.Validate().VarCtx(ctx, dirpath, "required,dirpath|filepath"); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	// Files
	form, err := c.MultipartForm()
	if err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	var resp []model.UploadFileResponse

	// TODO: add workerpool to handle multiple file uploads concurrently
	files := form.File["files"]
	for _, file := range files {
		// open file
		src, err := file.Open()
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
		defer src.Close()

		fullName := filepath.Join(identity.ID, dirpath, file.Filename)

		// save files
		size, err := s.FileService.CreateFile(ctx, src, fullName)
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		resp = append(resp, model.UploadFileResponse{
			Name: file.Filename,
			Size: size,
		})
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
// @Failure 500 {object} model.ErrorResponse
// @Router /files [get]
func (s *Server) ListEntries(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.ListEntriesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	// Identity ID will be used as root directory
	identity, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	fullPath := filepath.Join(string(filepath.Separator), identity.ID, req.DirPath) + string(filepath.Separator)
	canView, err := s.PermissionService.CanViewDirectory(ctx, identity.ID, fullPath)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(errors.New("not permitted to view")))
	}

	cursor := pagination.NewCursor(req.Cursor, req.Limit)
	files, err := s.FileService.ListEntries(ctx, fullPath, cursor)
	if err != nil {
		if errors.Is(err, file.ErrInvalidCursor) {
			return s.error(c, apperror.ErrInvalidParam(err))
		}

		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, model.ListEntriesResponse{
		Entries: files,
		Cursor:  cursor.NextToken(),
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
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/directories [post]
func (s *Server) CreateDirectory(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.CreateDirectoryRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	// Identity ID will be used as root directory
	identity, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	fullPath := filepath.Join(string(filepath.Separator), identity.ID, req.DirPath) + string(filepath.Separator)

	parent := filepath.Join(fullPath, "..") + string(filepath.Separator)
	canEdit, err := s.PermissionService.CanEditDirectory(ctx, identity.ID, parent)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(errors.New("not permitted to edit")))
	}

	if err := s.FileService.CreateDirectory(ctx, fullPath); err != nil && !errors.Is(err, file.ErrDirAlreadyExists) {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.PermissionService.CreateDirectoryPermissions(ctx, identity.ID, fullPath); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, nil)
}

func (s *Server) RegisterFileRoutes(router *echo.Group) {
	router.Use(s.passwordChangedAtMiddleware)
	router.POST("", s.UploadFiles)
	router.GET("", s.ListEntries)
	router.POST("/directories", s.CreateDirectory)
	router.GET("/metadata", s.GetMetadata)
	router.GET("/download", s.DownloadFile)
}

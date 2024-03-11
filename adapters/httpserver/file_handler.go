package httpserver

import (
	"errors"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"net/http"
	"path/filepath"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/mycontext"
	"github.com/SeaCloudHub/backend/pkg/validation"

	"github.com/labstack/echo/v4"
)

func (s *Server) GetFile(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.GetFileRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	f, err := s.FileService.GetFile(ctx, filepath.Join(id.ID, req.FilePath))
	if err != nil {
		if errors.Is(err, file.ErrFileNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, f)
}

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
		if errors.Is(err, file.ErrFileNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	return c.Stream(http.StatusOK, mime, f)
}

func (s *Server) UploadFiles(c echo.Context) error {
	var ctx = mycontext.NewEchoContextAdapter(c)

	// Identity ID will be used as root directory
	identity, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	// Directory
	dirpath := c.FormValue("dirpath")
	if err := validation.Validate().VarCtx(ctx, dirpath, "required,dirpath"); err != nil {
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
		size, err := s.FileService.CreateFile(ctx, src, fullName, file.Size)
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

	files, next, err := s.FileService.ListEntries(ctx, filepath.Join(identity.ID, req.DirPath), req.Limit, req.Cursor)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, model.ListEntriesResponse{
		Entries: files,
		Cursor:  next,
	})
}

func (s *Server) RegisterFileRoutes(router *echo.Group) {
	router.Use(s.passwordChangedAtMiddleware)
	router.POST("", s.UploadFiles)
	router.GET("", s.ListEntries)
	router.GET("/metadata", s.GetFile)
	router.GET("/download", s.DownloadFile)
}

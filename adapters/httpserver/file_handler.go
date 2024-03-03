package httpserver

import (
	"errors"
	"net/http"
	"path/filepath"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
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
		return s.handleError(c, err, http.StatusBadRequest)
	}

	if err := req.Validate(ctx); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	id, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	file, err := s.FileService.GetFile(ctx, filepath.Join(id.ID, req.FilePath))
	if err != nil {
		return s.handleError(c, err, http.StatusInternalServerError)
	}

	return s.success(c, file)
}

func (s *Server) UploadFiles(c echo.Context) error {
	var ctx = mycontext.NewEchoContextAdapter(c)

	// Identity ID will be used as root directory
	identity, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	// Directory
	dirpath := c.FormValue("dirpath")
	if err := validation.Validate().VarCtx(ctx, dirpath, "required,dirpath"); err != nil {
		return s.handleError(c, errors.New("invalid dirpath"), http.StatusBadRequest)
	}

	// Files
	form, err := c.MultipartForm()
	if err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	var resp []model.UploadFileResponse

	// TODO: add workerpool to handle multiple file uploads concurrently
	files := form.File["files"]
	for _, file := range files {
		// open file
		src, err := file.Open()
		if err != nil {
			return s.handleError(c, err, http.StatusInternalServerError)
		}
		defer src.Close()

		fullName := filepath.Join(identity.ID, dirpath, file.Filename)

		// save files
		size, err := s.FileService.CreateFile(ctx, src, fullName, file.Size)
		if err != nil {
			return s.handleError(c, err, http.StatusInternalServerError)
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
		return s.handleError(c, err, http.StatusBadRequest)
	}

	if err := req.Validate(ctx); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	// Identity ID will be used as root directory
	identity, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	files, next, err := s.FileService.ListEntries(ctx, filepath.Join(identity.ID, req.DirPath), req.Limit, req.Cursor)
	if err != nil {
		return s.handleError(c, err, http.StatusInternalServerError)
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
}

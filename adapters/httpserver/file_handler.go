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

//
//func (s *Server) GetFile(c echo.Context) error {
//	filename := c.Param("filename")
//	file, err := s.FileService.GetFile(filename)
//	if err != nil {
//		return s.handleError(c, err, http.StatusInternalServerError)
//	}
//	defer file.Close()
//
//	return c.Stream(http.StatusOK, "image/webp", file)
//}

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
		if _, err := s.FileService.CreateFile(ctx, src, fullName, file.Size); err != nil {
			return s.handleError(c, err, http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, "files uploaded")
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
	//router.GET(":filename", s.GetFile)
}

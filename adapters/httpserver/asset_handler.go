package httpserver

import (
	"mime"
	"path/filepath"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/pkg/app"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/labstack/echo/v4"
)

// GetImage godoc
// @Summary GetImage
// @Description GetImage
// @Tags assets
// @Param name path string true "Image name"
// @Success 200 {file} file
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /assets/images/{name} [get]
func (s *Server) GetImage(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.GetImageRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	reader, contentType, err := s.FileService.DownloadFile(ctx, filepath.Join("/assets", "images", req.Name))
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return c.Stream(200, contentType, reader)

}

// UploadImage godoc
// @Summary UploadImage
// @Description UploadImage
// @Tags assets
// @Accept multipart/form-data
// @Param image formData file true "Image file"
// @Success 200 {object} model.SuccessResponse{data=[]model.UploadImageResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /assets/images [post]
func (s *Server) UploadImage(c echo.Context) error {
	var ctx = app.NewEchoContextAdapter(c)

	reader, contentType, err := app.BindMultipartFile(c, "image")
	if err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if file.IsImage(contentType) {
		return s.error(c, apperror.ErrInvalidParam(file.ErrNotAnImage))
	}

	exts, _ := mime.ExtensionsByType(contentType)
	if len(exts) == 0 {
		return s.error(c, apperror.ErrInvalidParam(file.ErrNotAnImage))
	}

	fileName := gonanoid.Must(11) + exts[0]
	fullPath := filepath.Join("/assets", "images", fileName)

	size, err := s.FileService.CreateFile(ctx, reader, fullPath, contentType)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, &model.UploadImageResponse{
		FileName: fileName,
		FilePath: filepath.Join("/api/assets/images", fileName),
		Size:     size,
		MimeType: contentType,
	})
}

func (s *Server) RegisterAssetRoutes(router *echo.Group) {
	router.GET("/images/:name", s.GetImage)
	router.POST("/images", s.UploadImage)
}

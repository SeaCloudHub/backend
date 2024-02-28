package httpserver

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) GetFile(c echo.Context) error {
	filename := c.Param("filename")
	file, err := s.FileService.GetFile(filename)
	if err != nil {
		return s.handleError(c, err, http.StatusInternalServerError)
	}
	defer file.Close()

	return c.Stream(http.StatusOK, "image/webp", file)
}

func (s *Server) CreateFile(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	// save files
	if err := s.FileService.UploadFile(file); err != nil {
		return s.handleError(c, err, http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, "file uploaded")
}

func (s *Server) RegisterFileRoutes(router *echo.Group) {
	router.GET(":filename", s.GetFile)
	router.POST("", s.CreateFile)
}

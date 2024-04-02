package httpserver

import (
	"errors"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/SeaCloudHub/backend/pkg/mycontext"
	"github.com/SeaCloudHub/backend/pkg/util"
	"github.com/labstack/echo/v4"
)

// TriggerCreateUserDirectory godoc
// @Summary TriggerCreateUserDirectory
// @Description TriggerCreateUserDirectory
// @Tags main
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Success 200 {object} model.SuccessResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /main/trigger/create-user-directory [post]
func (s *Server) TriggerCreateUserDirectory(c echo.Context) error {
	ctx := mycontext.NewEchoContextAdapter(c)
	identities, _, err := s.IdentityService.ListIdentities(ctx, "", 0)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	for _, identity := range identities {
		dirPath := util.GetIdentityDirPath(identity.ID)
		_, err := s.FileService.GetMetadata(ctx, dirPath)
		if err != nil {
			if errors.Is(err, file.ErrNotFound) {
				err := s.FileService.CreateDirectory(ctx, dirPath)
				if err != nil {
					return s.error(c, apperror.ErrInternalServer(err))
				}
			} else {
				return s.error(c, apperror.ErrInternalServer(err))
			}
		}
	}

	return s.success(c, nil)
}

func (s *Server) RegisterMainRoutes(router *echo.Group) {
	router.POST("/trigger/create-user-directory",
		s.TriggerCreateUserDirectory, s.adminMiddleware)
}

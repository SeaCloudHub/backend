package httpserver

import (
	"errors"
	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/SeaCloudHub/backend/pkg/mycontext"

	"github.com/labstack/echo/v4"
)

// Login godoc
// @Summary Login
// @Description Login
// @Tags user
// @Accept json
// @Produce json
// @Param payload body model.LoginRequest true "Login request"
// @Success 200 {object} model.SuccessResponse{data=model.LoginResponse}
// @Failure 400 {object} model.ErrorResponse
// @Router /users/login [post]
func (s *Server) Login(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.LoginRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	session, err := s.IdentityService.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, identity.ErrInvalidCredentials) {
			return s.error(c, apperror.ErrInvalidCredentials(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, model.LoginResponse{SessionToken: *session.Token})
}

// Me godoc
// @Summary Me
// @Description Me
// @Tags user
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Success 200 {object} model.SuccessResponse{data=identity.Identity}
// @Failure 401 {object} model.ErrorResponse
// @Router /users/me [get]
func (s *Server) Me(c echo.Context) error {
	return s.success(c, c.Get(ContextKeyIdentity))
}

// ChangePassword godoc
// @Summary Change password
// @Description Change password
// @Tags user
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.ChangePasswordRequest true "Change password request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /users/change-password [post]
func (s *Server) ChangePassword(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.ChangePasswordRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, ok := c.Get(ContextKeyIdentity).(*identity.Identity)
	if !ok {
		return s.error(c, apperror.ErrInternalServer(errors.New("identity not found")))
	}

	if err := s.IdentityService.ChangePassword(ctx, id, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, identity.ErrInvalidCredentials) {
			return s.error(c, apperror.ErrInvalidCredentials(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.IdentityService.SyncPasswordChangedAt(ctx, id); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, nil)
}

func (s *Server) RegisterUserRoutes(router *echo.Group) {
	router.POST("/login", s.Login)
	router.GET("/me", s.Me)
	router.POST("/change-password", s.ChangePassword)
}

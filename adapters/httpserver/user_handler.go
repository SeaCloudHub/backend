package httpserver

import (
	"errors"
	"net/http"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/mycontext"

	"github.com/labstack/echo/v4"
)

func (s *Server) Login(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.LoginRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	if err := req.Validate(); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	session, err := s.IdentityService.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, identity.ErrInvalidCredentials) {
			return s.handleError(c, err, http.StatusBadRequest)
		}

		return s.handleError(c, err, http.StatusInternalServerError)
	}

	return s.success(c, model.LoginResponse{SessionToken: *session.Token})
}

func (s *Server) Me(c echo.Context) error {
	return c.JSON(http.StatusOK, c.Get(ContextKeyIdentity))
}

func (s *Server) ChangePassword(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.ChangePasswordRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	if err := req.Validate(); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	id, ok := c.Get(ContextKeyIdentity).(*identity.Identity)
	if !ok {
		return s.handleError(c, errors.New("identity not found"), http.StatusInternalServerError)
	}

	if err := s.IdentityService.ChangePassword(ctx, id, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, identity.ErrInvalidCredentials) {
			return s.handleError(c, err, http.StatusBadRequest)
		}

		return s.handleError(c, err, http.StatusInternalServerError)
	}

	if err := s.IdentityService.SyncPasswordChangedAt(ctx, id); err != nil {
		return s.handleError(c, err, http.StatusInternalServerError)
	}

	return s.success(c, "Password changed")
}

func (s *Server) RegisterUserRoutes(router *echo.Group) {
	router.POST("/login", s.Login)

	router.GET("/me", s.Me)
	router.POST("/change-password", s.ChangePassword)
}

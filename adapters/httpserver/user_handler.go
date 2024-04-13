package httpserver

import (
	"errors"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/app"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/google/uuid"

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
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /users/login [post]
func (s *Server) Login(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
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

		if errors.Is(err, identity.ErrIdentityWasDisabled) {
			return s.error(c, apperror.ErrIdentityWasDisabled(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	// get user from db
	user, err := s.UserStore.GetByID(ctx, uuid.MustParse(session.Identity.ID))
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	isAdmin, err := s.PermissionService.IsAdmin(ctx, session.Identity.ID)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if user.IsAdmin != isAdmin {
		// update user is_admin
		if err := s.UserStore.UpdateAdmin(ctx, uuid.MustParse(session.Identity.ID)); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	user.IsAdmin = isAdmin

	// update last login
	if err := s.UserStore.UpdateLastSignInAt(ctx, uuid.MustParse(session.Identity.ID)); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, model.LoginResponse{
		SessionToken:     *session.Token,
		SessionID:        session.ID,
		SessionExpiresAt: session.ExpiresAt,
		Identity:         user,
	})
}

// Me godoc
// @Summary Me
// @Description Me
// @Tags user
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Success 200 {object} model.SuccessResponse{data=identity.User}
// @Failure 401 {object} model.ErrorResponse
// @Router /users/me [get]
func (s *Server) Me(c echo.Context) error {
	return s.success(c, c.Get(ContextKeyUser))
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
		ctx = app.NewEchoContextAdapter(c)
		req model.ChangePasswordRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	if err := s.IdentityService.ChangePassword(ctx, id, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, identity.ErrIncorrectPassword) {
			return s.error(c, apperror.ErrIncorrectPassword(err))
		}

		if errors.Is(err, identity.ErrInvalidPassword) {
			return s.error(c, apperror.ErrInvalidPassword(err))
		}

		if errors.Is(err, identity.ErrSessionTooOld) {
			return s.error(c, apperror.ErrSessionRefreshRequired(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.UserStore.UpdatePasswordChangedAt(ctx, uuid.MustParse(id.ID)); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, nil)
}

// GetByEmail godoc
// @Summary Get user by email
// @Description Get user by email
// @Tags user
// @Produce json
// @Param email query string true "Email"
// @Success 200 {object} model.SuccessResponse{data=model.GetByEmailResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /users/email [get]
func (s *Server) GetByEmail(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.GetByEmailRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, err := s.UserStore.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, identity.ErrIdentityNotFound) {
			return s.error(c, apperror.ErrIdentityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, model.GetByEmailResponse{
		Email:             user.Email,
		FirstName:         user.FirstName,
		LastName:          user.LastName,
		AvatarURL:         user.AvatarURL,
		PasswordChangedAt: user.PasswordChangedAt,
	})
}

func (s *Server) RegisterUserRoutes(router *echo.Group) {
	router.POST("/login", s.Login)
	router.GET("/me", s.Me)
	router.POST("/change-password", s.ChangePassword)
	router.GET("/email", s.GetByEmail)
}

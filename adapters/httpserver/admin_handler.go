package httpserver

import (
	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	_ "github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/SeaCloudHub/backend/pkg/mycontext"

	"github.com/labstack/echo/v4"
)

// AdminMe godoc
// @Summary AdminMe
// @Description AdminMe
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Success 200 {object} model.SuccessResponse{data=identity.Identity}
// @Failure 401 {object} model.ErrorResponse
// @Router /admin/me [get]
func (s *Server) AdminMe(c echo.Context) error {
	return s.success(c, c.Get(ContextKeyIdentity))
}

// ListIdentities godoc
// @Summary ListIdentities
// @Description ListIdentities
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param pageToken query string false "Page token"
// @Param pageSize query int false "Page size"
// @Success 200 {object} model.SuccessResponse{data=model.ListIdentitiesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities [get]
func (s *Server) ListIdentities(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.ListIdentitiesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	identities, nextToken, err := s.IdentityService.ListIdentities(ctx, req.PageToken, req.PageSize)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, model.ListIdentitiesResponse{
		Identities: identities,
		NextToken:  nextToken,
	})
}

// CreateIdentity godoc
// @Summary CreateIdentity
// @Description CreateIdentity
// @Tags admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.CreateIdentityRequest true "Create identity request"
// @Success 200 {object} model.SuccessResponse{data=identity.Identity}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities [post]
func (s *Server) CreateIdentity(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.CreateIdentityRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, err := s.IdentityService.CreateIdentity(ctx, req.Email, req.Password)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	id.Password = req.Password

	return s.success(c, id)
}

func (s *Server) RegisterAdminRoutes(router *echo.Group) {
	router.Use(s.adminMiddleware)
	router.GET("/me", s.AdminMe)

	router.Use(s.passwordChangedAtMiddleware)
	router.GET("/identities", s.ListIdentities)
	router.POST("/identities", s.CreateIdentity)
}

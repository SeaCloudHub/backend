package httpserver

import (
	"errors"
	"net/http"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/mycontext"

	"github.com/labstack/echo/v4"
)

func (s *Server) AdminMe(c echo.Context) error {
	return s.success(c, c.Get(ContextKeyIdentity))
}

func (s *Server) ListIdentities(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.ListIdentitiesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	if err := req.Validate(); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	identities, nextToken, err := s.IdentityService.ListIdentities(ctx, req.PageToken, req.PageSize)
	if err != nil {
		return s.handleError(c, err, http.StatusInternalServerError)
	}

	return s.success(c, model.ListIdentitiesResponse{
		Identities: identities,
		NextToken:  nextToken,
	})
}

func (s *Server) CreateIdentity(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.CreateIdentityRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	if err := req.Validate(); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	id, err := s.IdentityService.CreateIdentity(ctx, req.Email, req.Password)
	if err != nil {
		return s.handleError(c, err, http.StatusInternalServerError)
	}

	id.Password = req.Password

	return s.success(c, id)
}

func (s *Server) RegisterAdminRoutes(router *echo.Group) {
	router.Use(s.adminMiddleware)
	router.GET("/me", s.AdminMe)
	router.GET("/identities", s.ListIdentities)
	router.POST("/identities", s.CreateIdentity)
}

func (s *Server) adminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			ctx = mycontext.NewEchoContextAdapter(c)
		)

		identity, ok := c.Get(ContextKeyIdentity).(*identity.Identity)
		if !ok {
			return s.handleError(c, errors.New("identity not found"), http.StatusInternalServerError)
		}

		isAdmin, err := s.PermissionService.IsManager(ctx, identity.ID)
		if err != nil {
			return s.handleError(c, err, http.StatusInternalServerError)
		}

		if !isAdmin {
			return s.handleError(c, errors.New("permission denied"), http.StatusForbidden)
		}

		return next(c)
	}
}

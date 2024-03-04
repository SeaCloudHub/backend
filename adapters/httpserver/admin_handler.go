package httpserver

import (
	"net/http"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/pkg/common"
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

func (s *Server) setStatusUserCommon(c echo.Context, state common.State) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req model.ChangeStateRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	if err := req.Validate(); err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	identitity, err := s.IdentityService.ChangeState(ctx, req.Id, state)
	if err != nil {
		return s.handleError(c, err, http.StatusInternalServerError)
	}

	return s.success(c, model.ChangeStateResponse{
		Identitiy: *identitity,
	})

}

func (s *Server) ActivateStateUser(c echo.Context) error {
	return s.setStatusUserCommon(c, common.ActiveState)

}

func (s *Server) DeActivateStateUser(c echo.Context) error {
	return s.setStatusUserCommon(c, common.DeActiveState)
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

	router.Use(s.passwordChangedAtMiddleware)
	router.GET("/identities", s.ListIdentities)
	router.POST("/identities", s.CreateIdentity)
	router.PATCH("/identities/:id/activate", s.ActivateStateUser)
	router.PATCH("/identities/:id/deactivate", s.DeActivateStateUser)
}

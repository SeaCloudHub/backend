package httpserver

import (
	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/pkg/mycontext"
	"net/http"

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

func (s *Server) CreateMultipleIdentities(c echo.Context) error {
	file, _, err := c.Request().FormFile("file")
	if err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}
	defer file.Close()

	var identities []*model.CreateIdentityRequest

	err = s.CSVService.CsvToEntities(&file, &identities)
	if err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	simpleIdentities, err := s.MapperService.ToIdentities(identities)
	if err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	ids, err := s.IdentityService.CreateMultipleIdentities(mycontext.NewEchoContextAdapter(c), simpleIdentities)
	if err != nil {
		return s.handleError(c, err, http.StatusInternalServerError)
	}

	for i := range ids {
		ids[i].Email = identities[i].Email
		ids[i].Password = identities[i].Password
	}

	return s.success(c, ids)
}

func (s *Server) RegisterAdminRoutes(router *echo.Group) {
	router.Use(s.adminMiddleware)
	router.GET("/me", s.AdminMe)

	router.Use(s.passwordChangedAtMiddleware)
	router.GET("/identities", s.ListIdentities)
	router.POST("/identities", s.CreateIdentity)
	router.POST("/identities/bulk", s.CreateMultipleIdentities)
}

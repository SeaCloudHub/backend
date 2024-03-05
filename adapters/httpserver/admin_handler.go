package httpserver

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
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

func (s *Server) CreateMultipleIdentities(c echo.Context) error {
	file, _, err := c.Request().FormFile("file")
	if err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}
	defer file.Close()

	entityMapper := func(record []string) interface{} {
		return model.CreateIdentityRequest{
			Email:    strings.TrimSpace(record[0]),
			Password: strings.TrimSpace(record[1]),
		}
	}

	records, err := s.CSVService.CsvToEntities(file, entityMapper)
	if err != nil {
		return s.handleError(c, err, http.StatusBadRequest)
	}

	var identities []model.CreateIdentityRequest
	for _, record := range records {
		identity, ok := record.(model.CreateIdentityRequest)
		if !ok {
			return s.handleError(c, fmt.Errorf("record is not of type *model.CreateIdentityRequest"), http.StatusBadRequest)
		}

		if err := identity.Validate(); err != nil {
			return s.handleError(c, err, http.StatusBadRequest)
		}

		identities = append(identities, identity)
	}

	ids, err := s.IdentityService.CreateMultipleIdentities(mycontext.NewEchoContextAdapter(c), s.MapperService.ToIdentities(identities))
	if err != nil {
		return s.handleError(c, err, http.StatusInternalServerError)
	}

	for i := range ids {
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

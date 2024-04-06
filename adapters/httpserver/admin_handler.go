package httpserver

import (
	"fmt"
	"github.com/SeaCloudHub/backend/adapters/event/listeners"
	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	_ "github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/SeaCloudHub/backend/pkg/mycontext"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/SeaCloudHub/backend/pkg/util"
	"github.com/labstack/echo/v4"
	"net/http"
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
// @Param paging query pagination.Paging false "Paging"
// @Success 200 {object} model.SuccessResponse{data=model.ListIdentitiesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities [get]
func (s *Server) ListIdentities(c echo.Context) error {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
		req pagination.Paging
	)
	// TODO: get maxCapacity from config of identity storage size
	// max capacity is 10GB for now
	const maxCapacity = 10 * 1024 * 1024 * 1024

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	identities, err := s.IdentityService.ListIdentities(ctx, &req)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}
	for i := range identities {
		identities[i].UsedCapacity, err = s.FileService.GetDirectorySize(ctx, util.GetIdentityDirPath(identities[i].ID))
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
		identities[i].MaximumCapacity = maxCapacity
	}

	return s.success(c, model.ListIdentitiesResponse{
		Identities: identities,
		Paging:     req,
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

	id, err := s.IdentityService.CreateIdentity(ctx,
		s.MapperService.ToIdentity(req), listeners.NewIdentityCreatedEventListener(s.FileService).EventHandler)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	id.Password = req.Password

	return s.success(c, id)
}

// CreateMultipleIdentities godoc
// @Summary CreateMultipleIdentities
// @Description CreateMultipleIdentities
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param file formData file true "CSV file"
// @Success 200 {object} model.SuccessResponse{data=[]identity.Identity}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /admin/identities/bulk [post]
func (s *Server) CreateMultipleIdentities(c echo.Context) error {
	file, _, err := c.Request().FormFile("file")
	if err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}
	defer file.Close()

	var identities []model.CreateIdentityRequest

	err = s.CSVService.CsvToEntities(&file, &identities)
	if err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	simpleIdentities, err := s.MapperService.ToIdentities(identities)
	if err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	ids, err := s.IdentityService.CreateMultipleIdentities(mycontext.
		NewEchoContextAdapter(c), simpleIdentities, listeners.NewIdentitiesPatchedListener(s.FileService).EventHandler)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	for i := range ids {
		ids[i].Email = identities[i].Email
		ids[i].Password = identities[i].Password
	}

	return s.success(c, ids)
}

// DownloadIdentitiesTemplate godoc
// @Summary Download Identities Template CSV
// @Description Download a CSV template file for creating identities.
// @Tags admin
// @Produce text/csv
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Success 200 {file} file "CSV file"
// @Failure 401 {object} model.ErrorResponse
// @Router /admin/identities/template [get]
func (s *Server) DownloadIdentitiesTemplate(c echo.Context) error {
	templateData := []model.CreateIdentityRequest{{}}

	buf, err := s.CSVService.EntitiesToCsv(templateData)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Set the headers
	c.Response().Header().Set(echo.HeaderContentType, "text/csv")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", "identities.csv"))

	return c.Blob(http.StatusOK, "text/csv", buf)
}

func (s *Server) RegisterAdminRoutes(router *echo.Group) {
	router.Use(s.adminMiddleware)
	router.GET("/me", s.AdminMe)

	router.Use(s.passwordChangedAtMiddleware)
	router.GET("/identities", s.ListIdentities)
	router.POST("/identities", s.CreateIdentity)
	router.POST("/identities/bulk", s.CreateMultipleIdentities)
	router.GET("/identities/template", s.DownloadIdentitiesTemplate)
}

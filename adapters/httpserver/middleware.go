package httpserver

import (
	"errors"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"strings"

	"github.com/SeaCloudHub/backend/pkg/mycontext"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	ContextKeyIdentity string = "identity"
)

type Authentication struct {
	SkipperPath []string
	KeyLookup   string
	AuthScheme  string

	server *Server
}

func (s *Server) NewAuthentication(keyLookup string, authScheme string, skipperPath []string) *Authentication {
	return &Authentication{
		SkipperPath: skipperPath,
		KeyLookup:   keyLookup,
		AuthScheme:  authScheme,
		server:      s,
	}
}

func (a *Authentication) Middleware() echo.MiddlewareFunc {
	skipper := func(c echo.Context) bool {
		return containFirst(a.SkipperPath, c.Path())
	}

	errorHandler := func(err error, c echo.Context) error {
		if skipper(c) {
			return nil
		}

		_ = a.server.error(c, apperror.ErrUnauthorized(err))

		return err
	}

	return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup:              a.KeyLookup,
		AuthScheme:             a.AuthScheme,
		Validator:              a.ValidateSessionToken,
		ErrorHandler:           errorHandler,
		ContinueOnIgnoredError: true,
	})
}

func (a *Authentication) ValidateSessionToken(token string, c echo.Context) (bool, error) {
	var (
		ctx = mycontext.NewEchoContextAdapter(c)
	)

	id, err := a.server.IdentityService.WhoAmI(ctx, token)
	if err != nil {
		return false, err
	}

	c.Set(ContextKeyIdentity, id)

	return true, nil
}

func (s *Server) adminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			ctx = mycontext.NewEchoContextAdapter(c)
		)

		id, ok := c.Get(ContextKeyIdentity).(*identity.Identity)
		if !ok {
			return s.error(c, apperror.ErrInternalServer(errors.New("identity not found")))
		}

		isAdmin, err := s.PermissionService.IsManager(ctx, id.ID)
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		if !isAdmin {
			return s.error(c, apperror.ErrForbidden(errors.New("not an admin")))
		}

		return next(c)
	}
}

func (s *Server) passwordChangedAtMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, ok := c.Get(ContextKeyIdentity).(*identity.Identity)
		if !ok {
			return s.error(c, apperror.ErrInternalServer(errors.New("identity not found")))
		}

		if id.PasswordChangedAt == nil {
			return s.error(c, apperror.ErrForbidden(errors.New("password not changed")))
		}

		return next(c)
	}
}

func containFirst(elems []string, v string) bool {
	for _, s := range elems {
		if strings.HasPrefix(v, s) {
			return true
		}
	}

	return false
}

package httpserver

import (
	"net/http"
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

		//logger.EchoContext(c).Error(err)

		_ = a.server.handleError(c, err, http.StatusUnauthorized)

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

	identity, err := a.server.IdentityService.WhoAmI(ctx, token)
	if err != nil {
		return false, err
	}

	c.Set(ContextKeyIdentity, identity)

	return true, nil
}

func containFirst(elems []string, v string) bool {
	for _, s := range elems {
		if strings.HasPrefix(v, s) {
			return true
		}
	}

	return false
}

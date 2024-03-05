package httpserver

import (
	"net/http"
	"strings"

	"github.com/SeaCloudHub/backend/domain/book"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/domain/permission"
	"github.com/SeaCloudHub/backend/internal"
	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/SeaCloudHub/backend/pkg/sentry"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

type Options func(s *Server) error

type Server struct {
	router *echo.Echo
	Config *config.Config
	Logger *zap.SugaredLogger

	// internal services
	MapperService internal.Mapper
	CSVService    internal.CSVService

	// storage adapters
	BookStore book.Storage

	// services
	FileService       file.Service
	IdentityService   identity.Service
	PermissionService permission.Service
}

func New(cfg *config.Config, logger *zap.SugaredLogger, options ...Options) (*Server, error) {
	s := Server{
		router: echo.New(),
		Config: cfg,
		Logger: logger,
	}

	for _, fn := range options {
		if err := fn(&s); err != nil {
			return nil, err
		}
	}

	s.RegisterGlobalMiddlewares()
	s.RegisterHealthCheck(s.router.Group(""))

	authMiddleware := s.NewAuthentication("header:Authorization", "Bearer",
		[]string{
			"/healthz",
			"/api/users/login"},
	).Middleware()

	s.router.Use(authMiddleware)

	s.RegisterBookRoutes(s.router.Group("/api/books"))
	s.RegisterUserRoutes(s.router.Group("/api/users"))
	s.RegisterAdminRoutes(s.router.Group("/api/admin"))
	s.RegisterFileRoutes(s.router.Group("/api/files"))

	return &s, nil
}

func (s *Server) RegisterGlobalMiddlewares() {
	s.router.Use(middleware.Recover())
	s.router.Use(middleware.Secure())
	s.router.Use(middleware.RequestID())
	s.router.Use(middleware.Gzip())
	s.router.Use(sentryecho.New(sentryecho.Options{Repanic: true}))

	// CORS
	if s.Config.AllowOrigins != "" {
		aos := strings.Split(s.Config.AllowOrigins, ",")
		s.router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: aos,
		}))
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) RegisterHealthCheck(router *echo.Group) {
	router.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK!!!")
	})
}

func (s *Server) handleError(c echo.Context, err error, status int) error {
	s.Logger.Errorw(
		err.Error(),
		zap.String("request_id", s.requestID(c)),
	)

	if status >= http.StatusInternalServerError {
		sentry.WithContext(c).Error(err)
	}

	return c.JSON(status, map[string]string{
		"message": http.StatusText(status),
		"info":    err.Error(),
	})
}

func (s *Server) success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "success",
		"data":    data,
	})
}

func (s *Server) requestID(c echo.Context) string {
	return c.Response().Header().Get(echo.HeaderXRequestID)
}

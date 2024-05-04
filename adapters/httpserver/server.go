package httpserver

import (
	"net/http"
	"strings"

	"github.com/SeaCloudHub/backend/domain"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	_ "github.com/SeaCloudHub/backend/docs"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/pkg/errors"
	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/domain/permission"
	"github.com/SeaCloudHub/backend/domain/pubsub"
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
	UserStore identity.Store
	FileStore file.Store

	// cache and stream adapters
	PubSubService pubsub.Service

	// services
	FileService       file.Service
	IdentityService   identity.Service
	PermissionService permission.Service

	// event bus
	EventDispatcher domain.EventDispatcher
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
	s.router.GET("/swagger/*", echoSwagger.WrapHandler)

	authMiddleware := s.NewAuthentication("header:Authorization", "Bearer",
		[]string{
			"/healthz",
			"/swagger",
			"/api/users/login",
			"/api/users/email",
			"/api/assets",
		},
	).Middleware()

	s.router.Use(authMiddleware)

	s.RegisterUserRoutes(s.router.Group("/api/users"))
	s.RegisterAdminRoutes(s.router.Group("/api/admin"))
	s.RegisterFileRoutes(s.router.Group("/api/files"))
	s.RegisterAssetRoutes(s.router.Group("/api/assets"))

	return &s, nil
}

func (s *Server) RegisterGlobalMiddlewares() {
	s.router.Use(middleware.Recover())
	s.router.Use(middleware.Secure())
	s.router.Use(middleware.RequestID())
	s.router.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: func(c echo.Context) bool { return strings.Contains(c.Request().URL.Path, "swagger") },
	}))
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

func (s *Server) error(c echo.Context, err error) error {
	s.Logger.Errorw(
		err.Error(),
		zap.String("request_id", s.requestID(c)),
	)

	var appErr apperror.Error
	if !errors.As(err, &appErr) {
		sentry.WithContext(c).Error(err)

		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    "000000",
			Message: "Internal Server Error",
			Info:    err.Error(),
		})
	}

	if appErr.HTTPCode >= http.StatusInternalServerError {
		sentry.WithContext(c).Error(err)
	}

	var errMessage string
	if appErr.Raw != nil {
		errMessage = appErr.Raw.Error()
	}

	return c.JSON(appErr.HTTPCode, model.ErrorResponse{
		Code:    appErr.ErrorCode,
		Message: appErr.Message,
		Info:    errMessage,
	})
}

func (s *Server) success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, model.SuccessResponse{
		Message: "OK",
		Data:    data,
	})
}

func (s *Server) requestID(c echo.Context) string {
	return c.Response().Header().Get(echo.HeaderXRequestID)
}

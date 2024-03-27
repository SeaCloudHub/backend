package main

import (
	"fmt"
	"github.com/SeaCloudHub/backend/adapters/event"
	"log"
	"net/http"

	"github.com/SeaCloudHub/backend/adapters/httpserver"
	"github.com/SeaCloudHub/backend/adapters/postgrestore"
	"github.com/SeaCloudHub/backend/adapters/services"
	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/SeaCloudHub/backend/pkg/logger"
	"github.com/SeaCloudHub/backend/pkg/sentry"
	sentrygo "github.com/getsentry/sentry-go"
	_ "github.com/lib/pq"
)

// @title SeaCloud APIs
// @version 1.0

// @BasePath /api
// @schemes http https

// @description Transaction API.
func main() {
	applog, err := logger.NewAppLogger()
	if err != nil {
		log.Fatalf("cannot load config: %v\n", err)
	}
	defer logger.Sync(applog)

	cfg, err := config.LoadConfig()
	if err != nil {
		applog.Fatal(err)
	}

	err = sentrygo.Init(sentrygo.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.AppEnv,
		AttachStacktrace: true,
	})
	if err != nil {
		applog.Fatalf("cannot init sentry: %v", err)
	}
	defer sentrygo.Flush(sentry.FlushTime)

	db, err := postgrestore.NewConnection(postgrestore.ParseFromConfig(cfg))
	if err != nil {
		applog.Fatal(err)
	}

	_ = db

	server, err := httpserver.New(cfg, applog)
	if err != nil {
		applog.Fatal(err)
	}

	// event bus
	server.EventDispatcher = event.NewEventDispatcher()

	// internal services
	server.CSVService = services.NewCSVService()
	server.MapperService = services.NewMapperService()


	server.FileService = services.NewFileService(cfg)
	server.IdentityService = services.NewIdentityService(cfg, server.EventDispatcher)
	server.PermissionService = services.NewPermissionService(cfg)

	addr := fmt.Sprintf(":%d", cfg.Port)
	applog.Info("server started!")
	applog.Fatal(http.ListenAndServe(addr, server))
}

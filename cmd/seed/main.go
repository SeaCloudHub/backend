package main

import (
	"context"
	"log"

	"github.com/SeaCloudHub/backend/adapters/services"
	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/SeaCloudHub/backend/pkg/logger"
	"github.com/SeaCloudHub/backend/pkg/sentry"
	sentrygo "github.com/getsentry/sentry-go"
	_ "github.com/lib/pq"
)

func main() {
	applog, err := logger.NewAppLogger()
	if err != nil {
		log.Fatalf("cannot load config: %v\n", err)
	}
	// defer logger.Sync(applog)

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

	identityService := services.NewIdentityService(cfg)
	permissionService := services.NewPermissionService(cfg)

	ctx := context.Background()
	email := "admin@seacloudhub.com"
	password := "plzdonthackme"

	// create admin user
	identity, err := identityService.CreateIdentity(ctx, email, password)
	if err != nil {
		applog.Fatalf("cannot create admin user: %v", err)
	}

	// create admin permission
	if err := permissionService.CreateManager(ctx, identity.ID); err != nil {
		applog.Fatalf("cannot create admin permission: %v", err)
	}

	applog.Info("admin user created successfully")
	applog.Infof("email: %s - password: %s", email, password)
}

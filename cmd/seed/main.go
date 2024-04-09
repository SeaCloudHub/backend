package main

import (
	"context"
	"log"

	"github.com/SeaCloudHub/backend/adapters/postgrestore"

	"github.com/SeaCloudHub/backend/adapters/services"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/SeaCloudHub/backend/pkg/logger"
	"github.com/SeaCloudHub/backend/pkg/sentry"
	"github.com/SeaCloudHub/backend/pkg/util"
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

	db, err := postgrestore.NewConnection(postgrestore.ParseFromConfig(cfg))
	if err != nil {
		applog.Fatalf("cannot connect to db: %v\n", err)
	}

	userStore := postgrestore.NewUserStore(db)
	identityService := services.NewIdentityService(cfg)
	permissionService := services.NewPermissionService(cfg)
	fileService := services.NewFileService(cfg)

	ctx := context.Background()

	// create admin user
	email := "admin@seacloudhub.com"
	password := "plzdonthackme"

	// create admin identity
	identity, err := identityService.CreateIdentity(ctx, identity.SimpleIdentity{
		Email:    email,
		Password: password,
	})
	if err != nil {
		applog.Fatalf("cannot create admin user: %v", err)
	}

	// create admin user
	user := identity.ToUser().WithName("Admin", "SeaCloudHub")
	if err := userStore.Create(ctx, user); err != nil {
		applog.Fatalf("cannot create admin user: %v", err)
	}

	// create admin permission
	if err := permissionService.CreateAdminGroup(ctx, identity.ID); err != nil {
		applog.Fatalf("cannot create admin permission: %v", err)
	}

	// update user to admin
	if err := userStore.UpdateAdmin(ctx, user.ID); err != nil {
		applog.Fatalf("cannot update user to admin: %v", err)
	}

	// create user root directory
	fullPath := util.GetIdentityDirPath(identity.ID)
	if err := fileService.CreateDirectory(ctx, fullPath); err != nil {
		applog.Fatalf("cannot create user root directory: %v", err)
	}

	// create user root directory permissions
	if err := permissionService.CreateDirectoryPermissions(ctx, identity.ID, fullPath); err != nil {
		applog.Fatalf("cannot create user root directory permissions: %v", err)
	}

	applog.Info("admin user created successfully")
	applog.Infof("email: %s - password: %s", email, password)
}

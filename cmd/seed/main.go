package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/SeaCloudHub/backend/adapters/postgrestore"
	"github.com/google/uuid"

	"github.com/SeaCloudHub/backend/adapters/services"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/domain/permission"
	"github.com/SeaCloudHub/backend/pkg/app"
	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/SeaCloudHub/backend/pkg/logger"
	"github.com/SeaCloudHub/backend/pkg/sentry"
	sentrygo "github.com/getsentry/sentry-go"
	_ "github.com/lib/pq"
)

type service struct {
	userStore         identity.Store
	fileStore         file.Store
	identityService   identity.Service
	permissionService permission.Service
	fileService       file.Service
}

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

	s := &service{
		userStore:         postgrestore.NewUserStore(db),
		fileStore:         postgrestore.NewFileStore(db),
		identityService:   services.NewIdentityService(cfg),
		permissionService: services.NewPermissionService(cfg),
		fileService:       services.NewFileService(cfg),
	}

	ctx := context.Background()

	// create admin user
	email := "admin@seacloudhub.com"
	password := "plzdonthackme"

	// create admin identity
	identity, err := s.identityService.CreateIdentity(ctx, identity.SimpleIdentity{
		Email:    email,
		Password: password,
	})
	if err != nil {
		applog.Fatalf("cannot create admin user: %v", err)
	}

	// create admin user
	user := identity.ToUser().WithName("Admin", "SeaCloudHub")
	if err := s.userStore.Create(ctx, user); err != nil {
		applog.Fatalf("cannot create admin user: %v", err)
	}

	// create admin permission
	if err := s.permissionService.CreateAdminGroup(ctx, identity.ID); err != nil {
		applog.Fatalf("cannot create admin permission: %v", err)
	}

	// update user to admin
	if err := s.userStore.UpdateAdmin(ctx, user.ID); err != nil {
		applog.Fatalf("cannot update user to admin: %v", err)
	}

	// create root directory
	rootID, err := s.createDirectory(ctx, user.ID, "/", "", "")
	if err != nil {
		applog.Fatalf("cannot create root directory: %v", err)
	}

	// create user root directory
	fullPath := app.GetIdentityDirPath(identity.ID)
	if err := s.fileService.CreateDirectory(ctx, fullPath); err != nil {
		applog.Fatalf("cannot create user root directory: %v", err)
	}

	userRootID, err := s.createDirectory(ctx, user.ID, fullPath, "/", rootID)
	if err != nil {
		applog.Fatalf("cannot create user root directory: %v", err)
	}

	// update user root id
	if err := s.userStore.UpdateRootID(ctx, user.ID, uuid.MustParse(userRootID)); err != nil {
		applog.Fatalf("cannot update user root id: %v", err)
	}

	// create user trash directory
	trashPath := filepath.Join(fullPath, ".trash") + string(filepath.Separator)
	if err := s.fileService.CreateDirectory(ctx, trashPath); err != nil {
		applog.Fatalf("cannot create user trash directory: %v", err)
	}

	if _, err := s.createDirectory(ctx, user.ID, trashPath, fullPath, userRootID); err != nil {
		applog.Fatalf("cannot create user trash directory: %v", err)
	}

	applog.Info("\nadmin user created successfully =======================================")
	applog.Infof("email: %s - password: %s", email, password)
}

func (s *service) createDirectory(ctx context.Context, ownerID uuid.UUID, fullPath string, path string, parentID string) (string, error) {
	// get metadata
	entry, err := s.fileService.GetMetadata(ctx, fullPath)
	if err != nil {
		return "", fmt.Errorf("get metadata: %w", err)
	}

	// create files row
	f := entry.ToFile().WithID(uuid.New()).WithPath(path).WithOwnerID(ownerID)
	if err := s.fileStore.Create(ctx, f); err != nil {
		return "", fmt.Errorf("create files row: %w", err)
	}

	// create directory permissions
	if err := s.permissionService.CreateDirectoryPermissions(ctx, ownerID.String(), f.ID.String(), parentID); err != nil {
		return "", fmt.Errorf("create directory permissions: %w", err)
	}

	return f.ID.String(), nil
}

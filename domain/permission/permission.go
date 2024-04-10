package permission

import "context"

type Service interface {
	IsAdmin(ctx context.Context, userID string) (bool, error)
	CreateAdminGroup(ctx context.Context, userID string) error
	CreateDirectoryPermissions(ctx context.Context, userID string, fullPath string) error
	CanEditDirectory(ctx context.Context, userID string, fullPath string) (bool, error)
	CanViewDirectory(ctx context.Context, userID string, fullPath string) (bool, error)
	CreateFilePermissions(ctx context.Context, userID string, fullPath string) error
	CanEditFile(ctx context.Context, userID string, fullPath string) (bool, error)
	CanViewFile(ctx context.Context, userID string, fullPath string) (bool, error)
}

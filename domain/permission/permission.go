package permission

import "context"

type Service interface {
	IsAdmin(ctx context.Context, userID string) (bool, error)
	CreateAdminGroup(ctx context.Context, userID string) error
	CreatePermission(ctx context.Context, in *CreatePermission) error
	CreateDirectoryPermissions(ctx context.Context, userID string, fileID string, parentID string) error
	CanEditDirectory(ctx context.Context, userID string, fileID string) (bool, error)
	CanViewDirectory(ctx context.Context, userID string, fileID string) (bool, error)
	ClearDirectoryPermissions(ctx context.Context, fileID string, userID string) error
	CreateFilePermissions(ctx context.Context, userID string, fileID string, parentID string) error
	CanEditFile(ctx context.Context, userID string, fileID string) (bool, error)
	CanViewFile(ctx context.Context, userID string, fileID string) (bool, error)
	ClearFilePermissions(ctx context.Context, fileID string, userID string) error
}

type CreatePermission struct {
	UserID    string
	FileID    string
	Namespace string // "Directory" or "File"
	Relation  string // "editors" or "viewers"
}

func NewCreatePermission(userID string, fileID string, isDir bool, role string) *CreatePermission {
	kind := "File"
	if isDir {
		kind = "Directory"
	}

	relation := "viewers"
	if role == "editor" {
		relation = "editors"
	}

	return &CreatePermission{
		UserID:    userID,
		FileID:    fileID,
		Namespace: kind,
		Relation:  relation,
	}
}

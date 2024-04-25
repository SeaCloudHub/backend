package permission

import (
	"context"
	"errors"
)

var (
	ErrNotPermittedToView   = errors.New("not permitted to view")
	ErrNotPermittedToEdit   = errors.New("not permitted to edit")
	ErrNotPermittedToDelete = errors.New("not permitted to delete")
)

var (
	RelationshipRoleMap = map[string]string{
		"owners":  "owner",
		"editors": "editor",
		"viewers": "viewer",
	}
	RoleRelationshipMap = map[string]string{
		"owner":  "owners",
		"editor": "editors",
		"viewer": "viewers",
	}
)

type Service interface {
	IsAdmin(ctx context.Context, userID string) (bool, error)
	CreateAdminGroup(ctx context.Context, userID string) error
	CreatePermission(ctx context.Context, in *CreatePermission) error
	GetSharedPermissions(ctx context.Context, userID string, namespace string, relation string) ([]string, error)

	CreateDirectoryPermissions(ctx context.Context, userID string, fileID string, parentID string) error
	CanEditDirectory(ctx context.Context, userID string, fileID string) (bool, error)
	CanViewDirectory(ctx context.Context, userID string, fileID string) (bool, error)
	CanDeleteDirectory(ctx context.Context, userID string, fileID string) (bool, error)
	IsDirectoryOwner(ctx context.Context, userID string, fileID string) (bool, error)
	ClearDirectoryPermissions(ctx context.Context, fileID string, userID string) error
	UpdateDirectoryParent(ctx context.Context, fileID string, parentID string, oldParentID string) error
	DeleteDirectoryPermissions(ctx context.Context, fileID string) error
	GetDirectoryUsers(ctx context.Context, fileID string) ([]FileUser, error)

	CreateFilePermissions(ctx context.Context, userID string, fileID string, parentID string) error
	CanEditFile(ctx context.Context, userID string, fileID string) (bool, error)
	CanViewFile(ctx context.Context, userID string, fileID string) (bool, error)
	CanDeleteFile(ctx context.Context, userID string, fileID string) (bool, error)
	IsFileOwner(ctx context.Context, userID string, fileID string) (bool, error)
	ClearFilePermissions(ctx context.Context, fileID string, userID string) error
	UpdateFileParent(ctx context.Context, fileID string, parentID string, oldParentID string) error
	DeleteFilePermissions(ctx context.Context, fileID string) error
	GetFileUsers(ctx context.Context, fileID string) ([]FileUser, error)
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

	relation := RoleRelationshipMap[role]

	return &CreatePermission{
		UserID:    userID,
		FileID:    fileID,
		Namespace: kind,
		Relation:  relation,
	}
}

type FileUser struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
} // @name permission.FileUser

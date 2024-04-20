package file

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/app"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/google/uuid"
)

type Store interface {
	Create(ctx context.Context, file *File) error
	ListPager(ctx context.Context, dirpath string, pager *pagination.Pager) ([]File, error)
	ListCursor(ctx context.Context, dirpath string, cursor *pagination.Cursor) ([]File, error)
	GetByID(ctx context.Context, id string) (*File, error)
	GetByFullPath(ctx context.Context, fullPath string) (*File, error)
	GetTrashByUserID(ctx context.Context, userID uuid.UUID) (*File, error)
	ListByIDs(ctx context.Context, ids []string) ([]File, error)
	ListSelected(ctx context.Context, parent *File, ids []string) ([]File, error)
	ListSelectedChildren(ctx context.Context, parent *File, ids []string) ([]File, error)
	ListSelectedOwnedChildren(ctx context.Context, userID uuid.UUID, parent *File, ids []string) ([]File, error)
	UpdateGeneralAccess(ctx context.Context, fileID uuid.UUID, generalAccess string) error
	UpdatePath(ctx context.Context, fileID uuid.UUID, name string, path string, fullPath string) error
	UpdateName(ctx context.Context, fileID uuid.UUID, name string) error
	MoveToTrash(ctx context.Context, fileID uuid.UUID, path string, fullPath string) error
	RestoreFromTrash(ctx context.Context, fileID uuid.UUID, path string, fullPath string) error
	RestoreChildrenFromTrash(ctx context.Context, parentPath, newPath string) ([]File, error)
	UpsertShare(ctx context.Context, fileID uuid.UUID, userIDs []uuid.UUID, role string) error
	GetShare(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (*Share, error)
	DeleteShare(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error
}

type File struct {
	ID            uuid.UUID   `json:"id"`
	Name          string      `json:"name"`
	Path          string      `json:"path"`
	FullPath      string      `json:"full_path"`
	ShownPath     string      `json:"shown_path"`
	PreviousPath  *string     `json:"-"`
	Size          uint64      `json:"size"`
	Mode          os.FileMode `json:"mode"`
	MimeType      string      `json:"mime_type"`
	MD5           []byte      `json:"md5"`
	IsDir         bool        `json:"is_dir"`
	GeneralAccess string      `json:"general_access"`
	OwnerID       uuid.UUID   `json:"owner_id"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`

	Owner *identity.User `json:"owner,omitempty"`
} // @name file.File

func (f *File) WithID(id uuid.UUID) *File {
	f.ID = id

	return f
}

func (f *File) WithName(name string) *File {
	f.Name = name

	return f
}

func (f *File) WithPath(path string) *File {
	if !strings.HasSuffix(path, "/") && len(path) > 0 {
		path = path + "/"
	}

	f.Path = path

	return f
}

func (f *File) WithFullPath(fullPath string) *File {
	f.FullPath = fullPath

	return f
}

func (f *File) WithOwnerID(ownerID uuid.UUID) *File {
	f.OwnerID = ownerID

	return f
}

func (f *File) Response() *File {
	f.ShownPath = app.RemoveRootPath(f.Path)

	return f
}

type Share struct {
	FileID    uuid.UUID `json:"file_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
} // @name file.Share

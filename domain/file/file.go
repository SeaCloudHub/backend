package file

import (
	"context"
	"os"
	"path/filepath"
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
	GetRootDirectory(ctx context.Context) (*File, error)
	GetTrashByUserID(ctx context.Context, userID uuid.UUID) (*File, error)
	ListByIDs(ctx context.Context, ids []string) ([]File, error)
	ListByFullPaths(ctx context.Context, fullPaths []string) ([]SimpleFile, error)
	ListSelected(ctx context.Context, parent *File, ids []string) ([]File, error)
	ListSelectedChildren(ctx context.Context, parent *File, ids []string) ([]File, error)
	ListSelectedOwnedChildren(ctx context.Context, userID uuid.UUID, parent *File, ids []string) ([]File, error)
	UpdateGeneralAccess(ctx context.Context, fileID uuid.UUID, generalAccess string) error
	UpdatePath(ctx context.Context, fileID uuid.UUID, path string) error
	UpdateName(ctx context.Context, fileID uuid.UUID, name string) error
	MoveToTrash(ctx context.Context, fileID uuid.UUID, path string) error
	RestoreFromTrash(ctx context.Context, fileID uuid.UUID, path string) error
	RestoreChildrenFromTrash(ctx context.Context, parentPath, newPath string) ([]File, error)
	Delete(ctx context.Context, file File) ([]File, error)
	UpsertShare(ctx context.Context, fileID uuid.UUID, userIDs []uuid.UUID, role string) error
	GetShare(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (*Share, error)
	DeleteShare(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error
	Star(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error
	Unstar(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error
	ListStarred(ctx context.Context, userID uuid.UUID) ([]File, error)
}

type File struct {
	ID            uuid.UUID   `json:"id"`
	Name          string      `json:"name"`
	Path          string      `json:"path"`
	ShownPath     string      `json:"shown_path"`
	PreviousPath  *string     `json:"-"`
	Size          uint64      `json:"size"`
	Mode          os.FileMode `json:"mode"`
	MimeType      string      `json:"mime_type"`
	Type          string      `json:"type"`
	MD5           []byte      `json:"md5"`
	IsDir         bool        `json:"is_dir"`
	GeneralAccess string      `json:"general_access"`
	OwnerID       uuid.UUID   `json:"owner_id"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`

	Owner *identity.User `json:"owner,omitempty"`
} // @name file.File

func NewDirectory(name string) *File {
	return &File{
		Name:     name,
		Size:     0,
		Mode:     os.ModeDir,
		MD5:      []byte{},
		MimeType: "",
		IsDir:    true,
	}
}

func (f *File) WithID(id uuid.UUID) *File {
	f.ID = id

	return f
}

func (f *File) WithName(name string) *File {
	f.Name = name

	return f
}

func (f *File) WithPath(path string) *File {
	f.Path = filepath.Clean(path)

	return f
}

func (f *File) WithOwnerID(ownerID uuid.UUID) *File {
	f.OwnerID = ownerID

	return f
}

func (f *File) FullPath() string {
	return filepath.Join(f.Path, f.Name)
}

func (f *File) Response() *File {
	f.ShownPath = app.RemoveRootPath(f.Path)

	return f
}

func (f *File) Parents() []string {
	if f.Path == "" || f.Path == "/" {
		return nil
	}

	// Initialize an empty slice to store parent paths
	var result []string

	currentPath := f.Path

	for currentPath != "" && currentPath != "/" {
		result = append(result, currentPath)

		currentPath = filepath.Dir(currentPath)
	}

	return result
}

type SimpleFile struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Path string    `json:"path"`
} // @name file.SimpleFile

type Share struct {
	FileID    uuid.UUID `json:"file_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
} // @name file.Share

type Stars struct {
	FileID    uuid.UUID `json:"file_id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
} // @name file.Stars

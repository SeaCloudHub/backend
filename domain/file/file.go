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
}

type File struct {
	ID        uuid.UUID   `json:"id"`
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	FullPath  string      `json:"full_path"`
	ShownPath string      `json:"shown_path"`
	Size      uint64      `json:"size"`
	Mode      os.FileMode `json:"mode"`
	MimeType  string      `json:"mime_type"`
	MD5       []byte      `json:"md5"`
	IsDir     bool        `json:"is_dir"`
	OwnerID   uuid.UUID   `json:"owner_id"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`

	Owner *identity.User `json:"owner,omitempty"`
} // @name file.File

func (f *File) WithID(id uuid.UUID) *File {
	f.ID = id

	return f
}

func (f *File) WithPath(path string) *File {
	if !strings.HasSuffix(path, "/") && len(path) > 0 {
		path = path + "/"
	}

	f.Path = path

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

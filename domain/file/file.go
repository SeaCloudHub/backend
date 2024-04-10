package file

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/SeaCloudHub/backend/pkg/util"
)

type Store interface {
	Create(ctx context.Context, file *File) error
	ListPager(ctx context.Context, dirpath string, pager *pagination.Pager) ([]File, error)
	ListCursor(ctx context.Context, dirpath string, cursor *pagination.Cursor) ([]File, error)
	GetByFullPath(ctx context.Context, fullPath string) (*File, error)
}

type File struct {
	ID        int         `json:"id"`
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	FullPath  string      `json:"full_path"`
	Size      uint64      `json:"size"`
	Mode      os.FileMode `json:"mode"`
	MimeType  string      `json:"mime_type"`
	MD5       []byte      `json:"md5"`
	IsDir     bool        `json:"is_dir"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
} // @name file.File

func (f *File) WithPath(path string) *File {
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	f.Path = path

	return f
}

func (f *File) RemoveRootPath() *File {
	f.FullPath = util.RemoveRootPath(f.FullPath)
	f.Path = util.RemoveRootPath(f.Path)

	return f
}

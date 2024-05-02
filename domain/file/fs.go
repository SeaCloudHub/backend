package file

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"
)

var (
	ErrNotFound         = errors.New("no such file or directory")
	ErrInvalidCursor    = errors.New("invalid cursor")
	ErrNotAnImage       = errors.New("only image file is allowed")
	ErrDirAlreadyExists = errors.New("directory already exists")
)

type Service interface {
	GetMetadata(ctx context.Context, id string) (*Entry, error)
	DownloadFile(ctx context.Context, id string) (io.ReadCloser, string, error)
	CreateFile(ctx context.Context, content io.Reader, id string, contentType string) (int64, error)
	Delete(ctx context.Context, id string) error
	DirStatus(ctx context.Context) (map[string]interface{}, error)
	VolStatus(ctx context.Context) (map[string]interface{}, error)
}

type Entry struct {
	Name      string      `json:"name"`
	FullPath  string      `json:"full_path"`
	Size      uint64      `json:"size"`
	Mode      os.FileMode `json:"mode"`
	MimeType  string      `json:"mime_type"`
	MD5       []byte      `json:"md5"`
	IsDir     bool        `json:"is_dir"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
} // @name file.Entry

func (e *Entry) ToFile(filename string) *File {
	return &File{
		Name:     filename,
		Size:     e.Size,
		Mode:     e.Mode,
		MimeType: e.MimeType,
		MD5:      e.MD5,
		IsDir:    e.IsDir,
	}
}

func IsImage(mimeType string) bool {
	return !strings.HasPrefix(mimeType, "image/")
}

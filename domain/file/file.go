package file

import (
	"context"
	"errors"
	"io"
	"os"
	"time"
)

var (
	ErrNotFound      = errors.New("no such file or directory")
	ErrInvalidCursor = errors.New("invalid cursor")
)

type Service interface {
	GetMetadata(ctx context.Context, fullPath string) (*Entry, error)
	DownloadFile(ctx context.Context, filePath string) (io.ReadCloser, string, error)
	CreateFile(ctx context.Context, content io.Reader, fullName string) (int64, error)
	ListEntries(ctx context.Context, dirpath string, limit int, cursor string) ([]Entry, string, error)
	CreateDirectory(ctx context.Context, dirpath string) error
	Delete(ctx context.Context, fullPath string) error
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

package services

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/SeaCloudHub/backend/pkg/seaweedfs"
	"github.com/pkg/errors"
)

type FileService struct {
	sw    *seaweedfs.Seaweed
	filer *seaweedfs.Filer
}

func NewFileService(cfg *config.Config) *FileService {
	swcfg := seaweedfs.NewConfigWithFilerURL(cfg.SeaweedFS.MasterServer, cfg.SeaweedFS.FilerServer)

	if cfg.DEBUG {
		swcfg = swcfg.Debug()
	}

	sw, err := seaweedfs.NewSeaweed(swcfg)
	if err != nil {
		panic(err)
	}

	return &FileService{
		sw:    sw,
		filer: sw.Filers()[0],
	}
}

func (s *FileService) GetMetadata(ctx context.Context, fullPath string) (*file.Entry, error) {
	resp, err := s.filer.GetMetadata(ctx, &seaweedfs.GetMetadataRequest{FullPath: fullPath})
	if err != nil {
		if errors.Is(err, seaweedfs.ErrNotFound) {
			return nil, file.ErrNotFound
		}

		return nil, fmt.Errorf("get metadata: %w", err)
	}

	entry := mapToEntry(resp)

	return &entry, nil
}

func (s *FileService) DownloadFile(ctx context.Context, filePath string) (io.Reader, string, error) {
	// entry, err := s.GetFile(ctx, filePath)
	// if err != nil {
	// 	return nil, "", err
	// }

	// if entry.IsDir {
	// 	return nil, "", errors.New("cannot download a directory")
	// }

	// var buf bytes.Buffer

	// if err := s.filer.Download(filePath, nil, func(reader io.Reader) error {
	// 	_, err := io.Copy(&buf, reader)
	// 	return err
	// }); err != nil {
	// 	return nil, "", err
	// }

	// return &buf, entry.MimeType, nil
	return nil, "", nil
}

func (s *FileService) CreateFile(_ context.Context, content io.Reader, fullName string, fileSize int64) (int64, error) {
	// result, err := s.filer.Upload(content, fileSize, fullName, "", "")
	// if err != nil {
	// 	return 0, err
	// }

	// return result.Size, nil
	return 0, nil
}

func (s *FileService) ListEntries(ctx context.Context, dirpath string, limit int, cursor string) ([]file.Entry, string, error) {
	// parse cursor
	cursorObj, err := pagination.DecodeCursor[swCursor](cursor)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %w", file.ErrInvalidCursor, err)
	}

	resp, err := s.filer.ListEntries(ctx, &seaweedfs.ListEntriesRequest{
		DirPath:      dirpath,
		Limit:        limit,
		LastFileName: cursorObj.LastFileName,
	})
	if err != nil {
		if errors.Is(err, seaweedfs.ErrNotFound) {
			return nil, "", file.ErrNotFound
		}

		return nil, "", fmt.Errorf("list entries: %w", err)
	}

	return handleListEntriesResponse(resp)
}

func (s *FileService) CreateDirectory(_ context.Context, dirpath string) error {
	return nil
}

func handleListEntriesResponse(resp *seaweedfs.ListEntriesResponse) ([]file.Entry, string, error) {
	entries := make([]file.Entry, 0, len(resp.Entries))
	for _, entry := range resp.Entries {
		entries = append(entries, mapToEntry(&entry))
	}

	cursor := ""
	if resp.ShouldDisplayLoadMore {
		cursor = pagination.EncodeCursor[swCursor](swCursor{LastFileName: resp.LastFileName})
	}

	return entries, cursor, nil
}

type swCursor struct {
	LastFileName string `json:"lastFileName"`
}

func mapToEntry(entry *seaweedfs.Entry) file.Entry {
	e := file.Entry{
		Name:      entry.FullPath.Name(),
		Size:      entry.FileSize,
		Mode:      entry.Mode,
		MimeType:  entry.Mime,
		MD5:       entry.Md5,
		IsDir:     entry.IsDirectory(),
		CreatedAt: entry.Crtime,
		UpdatedAt: entry.Mtime,
	}

	// remove the root path from the full path
	entryPath := entry.FullPath.Split()
	entryPath[0] = "/"
	e.FullPath = filepath.ToSlash(filepath.Join(entryPath...))

	return e
}

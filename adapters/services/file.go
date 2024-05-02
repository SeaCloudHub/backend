package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/SeaCloudHub/backend/pkg/seaweedfs"
)

type FileService struct {
	sw    *seaweedfs.Seaweed
	filer *seaweedfs.Filer
}

func NewFileService(cfg *config.Config) *FileService {
	swcfg := seaweedfs.NewConfigWithFilerURL(cfg.SeaweedFS.MasterServer, cfg.SeaweedFS.FilerServer)

	if cfg.Debug {
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

func (s *FileService) GetMetadata(ctx context.Context, id string) (*file.Entry, error) {
	resp, err := s.filer.GetMetadata(ctx, &seaweedfs.GetMetadataRequest{FullPath: filepath.Join("/", id)})
	if err != nil {
		if errors.Is(err, seaweedfs.ErrNotFound) {
			return nil, file.ErrNotFound
		}

		return nil, fmt.Errorf("get metadata: %w", err)
	}

	entry := mapToEntry(resp)

	return &entry, nil
}

func (s *FileService) DownloadFile(ctx context.Context, id string) (io.ReadCloser, string, error) {
	entry, err := s.GetMetadata(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("get metadata: %w", err)
	}

	if entry.IsDir {
		return nil, "", file.ErrNotFound
	}

	rc, err := s.filer.DownloadFile(ctx, &seaweedfs.DownloadFileRequest{FullPath: filepath.Join("/", id)})
	if err != nil {
		return nil, "", fmt.Errorf("download file: %w", err)
	}

	return rc, entry.MimeType, nil
}

func (s *FileService) CreateFile(ctx context.Context, content io.Reader, id string, contentType string) (int64, error) {
	result, err := s.filer.UploadFile(ctx, &seaweedfs.UploadFileRequest{
		Content:      content,
		FullFileName: filepath.Join("/", id),
		ContentType:  contentType,
	})
	if err != nil {
		return 0, err
	}

	return result.Size, nil
}

func (s *FileService) Delete(ctx context.Context, id string) error {
	err := s.filer.Delete(ctx, &seaweedfs.DeleteRequest{FullPath: filepath.Join("/", id)})
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

func (s *FileService) DirStatus(ctx context.Context) (map[string]interface{}, error) {
	return s.sw.Master().DirStatus(ctx)
}

func (s *FileService) VolStatus(ctx context.Context) (map[string]interface{}, error) {
	return s.sw.Master().VolStatus(ctx)
}

func mapToEntry(entry *seaweedfs.Entry) file.Entry {
	e := file.Entry{
		Name:      entry.FullPath.Name(),
		FullPath:  string(entry.FullPath),
		Size:      entry.FileSize,
		Mode:      entry.Mode,
		MimeType:  entry.Mime,
		MD5:       entry.Md5,
		IsDir:     entry.IsDirectory(),
		CreatedAt: entry.Crtime,
		UpdatedAt: entry.Mtime,
	}

	return e
}

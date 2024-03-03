package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/linxGnu/goseaweedfs"
	"github.com/seaweedfs/seaweedfs/weed/filer"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/pkg/config"
)

type FileService struct {
	sw    *goseaweedfs.Seaweed
	filer *goseaweedfs.Filer
}

func NewFileService(cfg *config.Config) *FileService {
	sw, err := goseaweedfs.NewSeaweed(cfg.SeaweedFS.MasterServer,
		[]string{cfg.SeaweedFS.FilerServer}, 8096, http.DefaultClient)
	if err != nil {
		panic(err)
	}

	return &FileService{
		sw:    sw,
		filer: sw.Filers()[0],
	}
}

//
//func (s *FileService) GetFile(filename string) (io.ReadCloser, error) {
//	req, err := http.NewRequest("GET", "/"+filename, nil)
//	if err != nil {
//		return nil, err
//	}
//
//	resp, err := s.client.Do(req)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.Body, nil
//}

func (s *FileService) CreateFile(_ context.Context, content io.Reader, fullName string, fileSize int64) (string, error) {
	result, err := s.filer.Upload(content, fileSize, fullName, "", "")
	if err != nil {
		return "", err
	}

	return result.FileID, nil
}

func (s *FileService) ListEntries(_ context.Context, dirpath string, limit int, cursor string) ([]file.Entry, string, error) {
	// parse cursor
	cursorObj, _ := decodeCursor(cursor)

	query := url.Values{}
	query.Set("limit", fmt.Sprintf("%d", limit))
	if cursorObj != nil && cursorObj.LastFileName != nil {
		query.Set("lastFileName", *cursorObj.LastFileName)
	}

	header := map[string]string{
		"Accept": "application/json",
	}

	data, code, err := s.filer.Get(dirpath, query, header)
	if err != nil {
		return nil, "", err
	}

	if code != http.StatusOK {
		return nil, "", errors.New("failed to list files and directories")
	}

	var resp listDirectoryEntriesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, "", err
	}

	return resp.mapToEntries(), resp.mapToCursor(), nil
}

type cursor struct {
	LastFileName *string
}

func newCursor(lastFileName string) *cursor {
	return &cursor{LastFileName: &lastFileName}
}

func (c *cursor) encode() string {
	data, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(data)
}

func decodeCursor(cursorStr string) (*cursor, error) {
	data, err := base64.StdEncoding.DecodeString(cursorStr)
	if err != nil {
		return nil, err
	}

	var cursorObj cursor
	if err := json.Unmarshal(data, &cursorObj); err != nil {
		return nil, err
	}

	return &cursorObj, nil
}

type listDirectoryEntriesResponse struct {
	Path                  string
	Entries               []filer.Entry
	Limit                 int
	LastFileName          string
	ShouldDisplayLoadMore bool
	EmptyFolder           bool
}

func (r *listDirectoryEntriesResponse) mapToEntries() []file.Entry {
	var entries []file.Entry
	for _, entry := range r.Entries {
		entries = append(entries, file.Entry{
			Name:      entry.FullPath.Name(),
			FullPath:  string(entry.FullPath),
			Size:      entry.FileSize,
			Mode:      entry.Mode,
			MimeType:  entry.Mime,
			MD5:       entry.Md5,
			IsDir:     entry.IsDirectory(),
			CreatedAt: entry.Crtime,
			UpdatedAt: entry.Mtime,
		})
	}

	return entries
}

func (r *listDirectoryEntriesResponse) mapToCursor() string {
	var cursor string
	if r.ShouldDisplayLoadMore {
		cursor = newCursor(r.LastFileName).encode()
	}

	return cursor
}

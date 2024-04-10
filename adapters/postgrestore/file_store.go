package postgrestore

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"gorm.io/gorm"
)

type FileStore struct {
	db *gorm.DB
}

func NewFileStore(db *gorm.DB) *FileStore {
	return &FileStore{db: db}
}

func (s *FileStore) Create(ctx context.Context, f *file.File) error {
	fileSchema := FileSchema{
		ID:       f.ID,
		Name:     f.Name,
		Path:     f.Path,
		FullPath: f.FullPath,
		Size:     f.Size,
		Mode:     uint32(fs.FileMode(f.Mode)),
		MimeType: f.MimeType,
		MD5:      hex.EncodeToString(f.MD5),
		IsDir:    f.IsDir,
	}

	if err := s.db.WithContext(ctx).Create(&fileSchema).Error; err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	f.ID = fileSchema.ID

	return nil
}

func (s *FileStore) ListPager(ctx context.Context, dirpath string, pager *pagination.Pager) ([]file.File, error) {
	var (
		fileSchemas []FileSchema
		total       int64
	)

	if err := s.db.WithContext(ctx).
		Where("path = ?", dirpath).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	pager.SetTotal(total)

	offset, limit := pager.Do()
	if err := s.db.WithContext(ctx).
		Where("path = ?", dirpath).
		Offset(offset).Limit(limit).Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	files := make([]file.File, len(fileSchemas))
	for i, fileSchema := range fileSchemas {
		md5, _ := hex.DecodeString(fileSchema.MD5)
		files[i] = file.File{
			ID:       fileSchema.ID,
			Name:     fileSchema.Name,
			Path:     fileSchema.Path,
			FullPath: fileSchema.FullPath,
			Size:     fileSchema.Size,
			Mode:     os.FileMode(fileSchema.Mode),
			MimeType: fileSchema.MimeType,
			MD5:      md5,
			IsDir:    fileSchema.IsDir,
		}
	}

	return files, nil
}

func (s *FileStore) ListCursor(ctx context.Context, dirpath string, cursor *pagination.Cursor) ([]file.File, error) {
	var fileSchemas []FileSchema

	// parse cursor
	cursorObj, err := pagination.DecodeToken[fsCursor](cursor.Token)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", file.ErrInvalidCursor, err)
	}

	if err := s.db.WithContext(ctx).
		Where("path = ?", dirpath).
		Where("id >= ?", cursorObj.ID).
		Limit(cursor.Limit + 1).Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	if len(fileSchemas) > cursor.Limit {
		cursor.SetNextToken(pagination.EncodeToken(fsCursor{ID: fileSchemas[cursor.Limit].ID}))
		fileSchemas = fileSchemas[:cursor.Limit]
	}

	files := make([]file.File, len(fileSchemas))
	for i, fileSchema := range fileSchemas {
		md5, _ := hex.DecodeString(fileSchema.MD5)
		files[i] = file.File{
			ID:        fileSchema.ID,
			Name:      fileSchema.Name,
			Path:      fileSchema.Path,
			FullPath:  fileSchema.FullPath,
			Size:      fileSchema.Size,
			Mode:      os.FileMode(fileSchema.Mode),
			MimeType:  fileSchema.MimeType,
			MD5:       md5,
			IsDir:     fileSchema.IsDir,
			CreatedAt: fileSchema.CreatedAt,
			UpdatedAt: fileSchema.UpdatedAt,
		}
	}

	return files, nil
}

func (s *FileStore) GetByFullPath(ctx context.Context, fullPath string) (*file.File, error) {
	var fileSchema FileSchema

	if err := s.db.WithContext(ctx).
		Where("full_path = ?", fullPath).
		First(&fileSchema).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, file.ErrNotFound
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	md5, _ := hex.DecodeString(fileSchema.MD5)
	return &file.File{
		ID:        fileSchema.ID,
		Name:      fileSchema.Name,
		Path:      fileSchema.Path,
		FullPath:  fileSchema.FullPath,
		Size:      fileSchema.Size,
		Mode:      os.FileMode(fileSchema.Mode),
		MimeType:  fileSchema.MimeType,
		MD5:       md5,
		IsDir:     fileSchema.IsDir,
		CreatedAt: fileSchema.CreatedAt,
		UpdatedAt: fileSchema.UpdatedAt,
	}, nil
}

type fsCursor struct {
	ID int `json:"id"`
}

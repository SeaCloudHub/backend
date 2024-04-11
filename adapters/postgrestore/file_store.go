package postgrestore

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"time"

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
		OwnerID:  f.OwnerID,
	}

	if err := s.db.WithContext(ctx).Create(&fileSchema).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return file.ErrDirAlreadyExists
		}

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

	if err := s.db.WithContext(ctx).Model(&fileSchemas).
		Where("path = ?", dirpath).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	pager.SetTotal(total)

	offset, limit := pager.Do()
	if err := s.db.WithContext(ctx).
		Preload("Owner").
		Where("path = ?", dirpath).
		Offset(offset).Limit(limit).Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	files := make([]file.File, len(fileSchemas))
	for i, fileSchema := range fileSchemas {
		files[i] = *fileSchema.ToDomainFile()
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

	query := s.db.WithContext(ctx).Where("path = ?", dirpath)
	if cursorObj.CreatedAt != nil {
		query = query.Where("created_at >= ?", cursorObj.CreatedAt)
	}

	if err := query.Limit(cursor.Limit + 1).Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	if len(fileSchemas) > cursor.Limit {
		cursor.SetNextToken(pagination.EncodeToken(fsCursor{CreatedAt: &fileSchemas[cursor.Limit].CreatedAt}))
		fileSchemas = fileSchemas[:cursor.Limit]
	}

	files := make([]file.File, len(fileSchemas))
	for i, fileSchema := range fileSchemas {
		files[i] = *fileSchema.ToDomainFile()
	}

	return files, nil
}

func (s *FileStore) GetByID(ctx context.Context, id string) (*file.File, error) {
	var fileSchema FileSchema

	if err := s.db.WithContext(ctx).
		Preload("Owner").
		Where("id = ?", id).
		First(&fileSchema).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, file.ErrNotFound
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return fileSchema.ToDomainFile(), nil
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

	return fileSchema.ToDomainFile(), nil
}

type fsCursor struct {
	CreatedAt *time.Time
}

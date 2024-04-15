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
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FileStore struct {
	db *gorm.DB
}

func NewFileStore(db *gorm.DB) *FileStore {
	return &FileStore{db: db}
}

func (s *FileStore) Create(ctx context.Context, f *file.File) error {
	fileSchema := FileSchema{
		ID:            f.ID,
		Name:          f.Name,
		Path:          f.Path,
		FullPath:      f.FullPath,
		Size:          f.Size,
		Mode:          uint32(fs.FileMode(f.Mode)),
		MimeType:      f.MimeType,
		MD5:           hex.EncodeToString(f.MD5),
		IsDir:         f.IsDir,
		GeneralAccess: "restricted",
		OwnerID:       f.OwnerID,
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, file.ErrNotFound
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return fileSchema.ToDomainFile(), nil
}

func (s *FileStore) ListByIDs(ctx context.Context, ids []string) ([]file.File, error) {
	var fileSchemas []FileSchema

	if err := s.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	files := make([]file.File, len(fileSchemas))
	for i, fileSchema := range fileSchemas {
		files[i] = *fileSchema.ToDomainFile()
	}

	return files, nil
}

func (s *FileStore) ListSelectedChildren(ctx context.Context, parent *file.File, ids []string) ([]file.File, error) {
	var fileSchemas []FileSchema

	if err := s.db.WithContext(ctx).
		Where("id IN ?", ids).
		Where("full_path LIKE ?", fmt.Sprintf("%s%%", parent.FullPath)).
		Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	files := make([]file.File, len(fileSchemas))
	for i, fileSchema := range fileSchemas {
		files[i] = *fileSchema.ToDomainFile()
	}

	return files, nil
}

func (s *FileStore) UpdateGeneralAccess(ctx context.Context, fileID uuid.UUID, generalAccess string) error {
	if err := s.db.WithContext(ctx).
		Model(&FileSchema{}).
		Where("id = ?", fileID).
		Update("general_access", generalAccess).Error; err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *FileStore) UpdatePath(ctx context.Context, fileID uuid.UUID, path, fullPath string) error {
	if err := s.db.WithContext(ctx).
		Model(&FileSchema{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"id":            fileID,
			"path":          path,
			"full_path":     fullPath,
			"previous_path": gorm.Expr("path"),
		}).Error; err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *FileStore) UpsertShare(ctx context.Context, fileID uuid.UUID, userIDs []uuid.UUID, role string) error {
	var shareSchemas []ShareSchema

	for _, userID := range userIDs {
		shareSchemas = append(shareSchemas, ShareSchema{
			FileID: fileID,
			UserID: userID,
			Role:   role,
		})
	}

	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "file_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role"}),
	}).Create(&shareSchemas).Error; err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *FileStore) GetShare(ctx context.Context, fileID, userID uuid.UUID) (*file.Share, error) {
	var shareSchema ShareSchema

	if err := s.db.WithContext(ctx).
		Where("file_id = ? AND user_id = ?", fileID, userID).
		First(&shareSchema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, file.ErrNotFound
		}
	}

	return &file.Share{
		FileID:    shareSchema.FileID,
		UserID:    shareSchema.UserID,
		Role:      shareSchema.Role,
		CreatedAt: shareSchema.CreatedAt,
	}, nil
}

func (s *FileStore) DeleteShare(ctx context.Context, fileID, userID uuid.UUID) error {
	if err := s.db.WithContext(ctx).
		Where("file_id = ? AND user_id = ?", fileID, userID).
		Delete(&ShareSchema{}).Error; err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

type fsCursor struct {
	CreatedAt *time.Time
}

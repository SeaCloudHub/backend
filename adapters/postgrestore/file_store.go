package postgrestore

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
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

func (s *FileStore) ListCursor(ctx context.Context, dirpath string, cursor *pagination.Cursor, filter file.Filter) ([]file.File, error) {
	var fileSchemas []FileSchema

	// parse cursor
	cursorObj, err := pagination.DecodeToken[fsCursor](cursor.Token)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", file.ErrInvalidCursor, err)
	}

	query := s.db.WithContext(ctx).Where("path = ?", dirpath).Where("name != ?", ".trash")
	if cursorObj.CreatedAt != nil {
		query = query.Where("created_at >= ?", cursorObj.CreatedAt)
	}

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	if filter.After != nil {
		query = query.Where("updated_at > ?", filter.After)
	}

	if err := query.Limit(cursor.Limit + 1).Order("created_at ASC").Order("id ASC").
		Find(&fileSchemas).Error; err != nil {
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

	path, name := filepath.Split(fullPath)

	if err := s.db.WithContext(ctx).
		Preload("Owner").
		Where("path = ?", filepath.Clean(path)).
		Where("name = ?", name).
		First(&fileSchema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, file.ErrNotFound
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return fileSchema.ToDomainFile(), nil
}

func (s *FileStore) GetRootDirectory(ctx context.Context) (*file.File, error) {
	var fileSchema FileSchema

	if err := s.db.WithContext(ctx).
		Where("name = ?", "/").
		First(&fileSchema).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return fileSchema.ToDomainFile(), nil
}

func (s *FileStore) GetTrashByUserID(ctx context.Context, userID uuid.UUID) (*file.File, error) {
	var fileSchema FileSchema

	if err := s.db.WithContext(ctx).
		Preload("Owner").
		Where("name = ?", ".trash").
		Where("owner_id = ?", userID).
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
		Preload("Owner").
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

func (s *FileStore) ListByFullPaths(ctx context.Context, fullPaths []string) ([]file.SimpleFile, error) {
	var (
		fileSchemas []FileSchema
		conditions  [][2]string
	)

	for _, fullPath := range fullPaths {
		path, name := filepath.Split(fullPath)
		conditions = append(conditions, [2]string{filepath.Clean(path), name})
	}

	if err := s.db.WithContext(ctx).
		Where("(path, name) IN ?", conditions).
		Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	files := make([]file.SimpleFile, len(fileSchemas))
	for i, fileSchema := range fileSchemas {
		files[i] = file.SimpleFile{
			ID:   fileSchema.ID,
			Name: fileSchema.Name,
			Path: fileSchema.Path,
		}
	}

	return files, nil
}

func (s *FileStore) ListSelected(ctx context.Context, parent *file.File, ids []string) ([]file.File, error) {
	var (
		fileSchemas []FileSchema
		files       []file.File
	)

	if err := s.db.WithContext(ctx).
		Where("id IN ?", ids).
		Where("path = ?", parent.FullPath()).
		Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	for _, fileSchema := range fileSchemas {
		files = append(files, *fileSchema.ToDomainFile())
	}

	return files, nil
}

func (s *FileStore) ListSelectedChildren(ctx context.Context, parent *file.File, ids []string) ([]file.File, error) {
	var (
		fileSchemas []FileSchema
		files       []file.File
	)

	db := s.db

	if err := db.WithContext(ctx).
		Where("id IN ?", ids).
		Where("path = ?", parent.FullPath()).
		Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	for _, fileSchema := range fileSchemas {
		files = append(files, *fileSchema.ToDomainFile())

		if !fileSchema.IsDir {
			continue
		}

		var childFileSchemas []FileSchema

		if err := db.WithContext(ctx).
			Where("path ~ ?", fmt.Sprintf(`^(%s(/.*)?)?$`, fileSchema.FullPath())).
			Find(&childFileSchemas).Error; err != nil {
			return nil, fmt.Errorf("unexpected error: %w", err)
		}

		for _, childFileSchema := range childFileSchemas {
			files = append(files, *childFileSchema.ToDomainFile())
		}
	}

	return files, nil
}

func (s *FileStore) ListSelectedOwnedChildren(ctx context.Context, userID uuid.UUID, parent *file.File, ids []string) ([]file.File, error) {
	var (
		fileSchemas []FileSchema
		files       []file.File
	)

	db := s.db

	if err := db.WithContext(ctx).
		Where("id IN ?", ids).
		Where("path = ?", parent.FullPath()).
		Where("owner_id = ?", userID).
		Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	for _, fileSchema := range fileSchemas {
		files = append(files, *fileSchema.ToDomainFile())

		if !fileSchema.IsDir {
			continue
		}

		var childFileSchemas []FileSchema

		if err := db.WithContext(ctx).
			Where("path ~ ?", fmt.Sprintf(`^(%s(/.*)?)?$`, fileSchema.FullPath())).
			Find(&childFileSchemas).Error; err != nil {
			return nil, fmt.Errorf("unexpected error: %w", err)
		}

		for _, childFileSchema := range childFileSchemas {
			files = append(files, *childFileSchema.ToDomainFile())
		}
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

func (s *FileStore) UpdatePath(ctx context.Context, fileID uuid.UUID, path string) error {
	if err := s.db.WithContext(ctx).
		Model(&FileSchema{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"id":   fileID,
			"path": path,
		}).Error; err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *FileStore) UpdateName(ctx context.Context, fileID uuid.UUID, name string) error {
	var fileSchema FileSchema

	// Retrieve file information
	if err := s.db.WithContext(ctx).Where("id = ?", fileID).First(&fileSchema).Error; err != nil {
		return fmt.Errorf("get file: %w", err)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update name and paths
		if err := tx.WithContext(ctx).Model(&FileSchema{}).Where("id = ?", fileID).
			Updates(map[string]interface{}{
				"name": name,
			}).Error; err != nil {
			return fmt.Errorf("update name: %w", err)
		}

		// Update child folders and file paths only if it's a folder
		if fileSchema.IsDir {
			if err := tx.WithContext(ctx).Model(&FileSchema{}).
				Where("path ~ ?", fmt.Sprintf(`^(%s(/.*)?)?$`, fileSchema.FullPath())).
				Updates(map[string]interface{}{
					"path": gorm.Expr("REPLACE(path, ?, ?)", fileSchema.FullPath(), filepath.Join(fileSchema.Path, name)),
				}).Error; err != nil {
				return fmt.Errorf("update child folders and files: %w", err)
			}
		}

		return nil
	})
}

func (s *FileStore) UpdateThumbnail(ctx context.Context, fileID uuid.UUID, thumbnail string) error {
	if err := s.db.WithContext(ctx).
		Model(&FileSchema{}).
		Where("id = ?", fileID).
		Update("thumbnail", thumbnail).Error; err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *FileStore) MoveToTrash(ctx context.Context, fileID uuid.UUID, path string) error {
	if err := s.db.WithContext(ctx).
		Model(&FileSchema{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"id":            fileID,
			"path":          path,
			"previous_path": gorm.Expr("path"),
		}).Error; err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *FileStore) RestoreFromTrash(ctx context.Context, fileID uuid.UUID, path string) error {
	if err := s.db.WithContext(ctx).
		Model(&FileSchema{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"id":            fileID,
			"path":          path,
			"previous_path": nil,
		}).Error; err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *FileStore) RestoreChildrenFromTrash(ctx context.Context, parentPath, newPath string) ([]file.File, error) {
	var fileSchemas []FileSchema

	if err := s.db.WithContext(ctx).
		Model(&fileSchemas).
		Clauses(clause.Returning{}).
		Where("path ~ ?", fmt.Sprintf(`^(%s(/.*)?)?$`, parentPath)).
		Updates(map[string]interface{}{
			"path":          gorm.Expr("replace(path, ?, ?)", parentPath, newPath),
			"previous_path": nil,
		}).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	var files []file.File
	for _, fileSchema := range fileSchemas {
		files = append(files, *fileSchema.ToDomainFile())
	}

	return files, nil
}

func (s *FileStore) Delete(ctx context.Context, e file.File) ([]file.File, error) {
	var (
		fileSchemas []FileSchema
		files       []file.File
	)

	if err := s.db.WithContext(ctx).Unscoped().
		Where("id = ?", e.ID).
		Delete(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	if e.IsDir {
		if err := s.db.WithContext(ctx).Unscoped().
			Clauses(clause.Returning{}).
			Where("path ~ ?", fmt.Sprintf(`^(%s(/.*)?)?$`, e.FullPath())).
			Delete(&fileSchemas).Error; err != nil {
			return nil, fmt.Errorf("unexpected error: %w", err)
		}

		for _, fileSchema := range fileSchemas {
			files = append(files, *fileSchema.ToDomainFile())
		}
	}

	return files, nil
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

func (s *FileStore) Star(ctx context.Context, fileID, userID uuid.UUID) error {
	var starSchema StarSchema

	if err := s.db.WithContext(ctx).
		Where("file_id = ? AND user_id = ?", fileID, userID).
		First(&starSchema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			starSchema = StarSchema{
				FileID: fileID,
				UserID: userID,
			}

			if err := s.db.WithContext(ctx).Create(&starSchema).Error; err != nil {
				return fmt.Errorf("unexpected error: %w", err)
			}
		}
	}

	return nil
}

func (s *FileStore) Unstar(ctx context.Context, fileID, userID uuid.UUID) error {
	if err := s.db.WithContext(ctx).
		Where("file_id = ? AND user_id = ?", fileID, userID).
		Delete(&StarSchema{}).Error; err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *FileStore) ListStarred(ctx context.Context, userID uuid.UUID) ([]file.File, error) {
	var fileSchemas []FileSchema

	if err := s.db.WithContext(ctx).
		Joins("JOIN stars ON files.id = stars.file_id").
		Where("stars.user_id = ?", userID).
		Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	files := make([]file.File, len(fileSchemas))
	for i, fileSchema := range fileSchemas {
		files[i] = *fileSchema.ToDomainFile()
	}

	return files, nil

}

func (s *FileStore) GetAllFiles(ctx context.Context) ([]file.File, error) {
	var fileSchemas []FileSchema

	if err := s.db.WithContext(ctx).
		Where("is_dir = ?", false).
		Find(&fileSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	files := make([]file.File, len(fileSchemas))
	for i, fileSchema := range fileSchemas {
		files[i] = *fileSchema.ToDomainFile()
	}

	return files, nil
}

type fsCursor struct {
	CreatedAt *time.Time
}

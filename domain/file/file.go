package file

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/app"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/google/uuid"
)

type Store interface {
	Create(ctx context.Context, file *File) error
	ListPager(ctx context.Context, dirpath string, pager *pagination.Pager) ([]File, error)
	ListCursor(ctx context.Context, dirpath string, cursor *pagination.Cursor, filter Filter) ([]File, error)
	Search(ctx context.Context, query string, cursor *pagination.Cursor, filter Filter) ([]File, error)
	GetByID(ctx context.Context, id string) (*File, error)
	GetByFullPath(ctx context.Context, fullPath string) (*File, error)
	GetRootDirectory(ctx context.Context) (*File, error)
	GetTrashByUserID(ctx context.Context, userID uuid.UUID) (*File, error)
	ListByIDs(ctx context.Context, ids []string) ([]File, error)
	ListByFullPaths(ctx context.Context, fullPaths []string) ([]SimpleFile, error)
	ListSelected(ctx context.Context, parent *File, ids []string) ([]File, error)
	ListSelectedChildren(ctx context.Context, parent *File, ids []string) ([]File, error)
	ListSelectedOwnedChildren(ctx context.Context, userID uuid.UUID, parent *File, ids []string) ([]File, error)
	ListFiles(ctx context.Context, path string, cursor *pagination.Cursor, filter Filter, asc bool) ([]File, error)
	UpdateGeneralAccess(ctx context.Context, fileID uuid.UUID, generalAccess string) error
	UpdatePath(ctx context.Context, fileID uuid.UUID, path string) error
	UpdateName(ctx context.Context, fileID uuid.UUID, name string) error
	UpdateThumbnail(ctx context.Context, fileID uuid.UUID, thumbnail string) error
	MoveToTrash(ctx context.Context, fileID uuid.UUID, path string) error
	RestoreFromTrash(ctx context.Context, fileID uuid.UUID, path string) error
	RestoreChildrenFromTrash(ctx context.Context, parentPath, newPath string) ([]File, error)
	Delete(ctx context.Context, file File) ([]File, error)
	UpsertShare(ctx context.Context, fileID uuid.UUID, userIDs []uuid.UUID, role string) error
	GetShare(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (*Share, error)
	DeleteShare(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error
	Star(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error
	Unstar(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error
	ListStarred(ctx context.Context, userID uuid.UUID) ([]File, error)
	GetAllFiles(ctx context.Context, path ...string) ([]File, error)
	ListRootDirectory(ctx context.Context, pager *pagination.Pager) ([]File, error)
	ListUserFiles(ctx context.Context, userID uuid.UUID) ([]*File, error)
	DeleteUserFiles(ctx context.Context, userID uuid.UUID) error
	DeleteShareByFileID(ctx context.Context, fileID uuid.UUID) error
	DeleteShareByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteStarByFileID(ctx context.Context, fileID uuid.UUID) error
	DeleteStarByUserID(ctx context.Context, userID uuid.UUID) error
	WriteLogs(ctx context.Context, logs []Log) error
	ReadLogs(ctx context.Context, userID string, cursor *pagination.Cursor) ([]Log, error)
	ListSuggested(ctx context.Context, userID uuid.UUID, limit int, isDir bool) ([]File, error)
	ListActivities(ctx context.Context, fileID uuid.UUID, cursor *pagination.Cursor) ([]Log, error)
}

type File struct {
	ID            uuid.UUID   `json:"id"`
	Name          string      `json:"name"`
	Path          string      `json:"path"`
	ShownPath     string      `json:"shown_path"`
	PreviousPath  *string     `json:"-"`
	Size          uint64      `json:"size"`
	Mode          os.FileMode `json:"mode"`
	MimeType      string      `json:"mime_type"`
	Type          string      `json:"type"`
	Thumbnail     *string     `json:"thumbnail"`
	MD5           []byte      `json:"md5"`
	IsDir         bool        `json:"is_dir"`
	GeneralAccess string      `json:"general_access"`
	OwnerID       uuid.UUID   `json:"owner_id"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`

	Owner  *identity.User `json:"owner,omitempty"`
	Parent *SimpleFile    `json:"parent,omitempty"`
	Log    *Log           `json:"log,omitempty"`
} // @name file.File

func NewDirectory(name string) *File {
	return &File{
		Name:     name,
		Size:     0,
		Mode:     os.ModeDir,
		MD5:      []byte{},
		MimeType: "",
		IsDir:    true,
	}
}

func (f *File) WithID(id uuid.UUID) *File {
	f.ID = id

	return f
}

func (f *File) WithName(name string) *File {
	f.Name = name

	return f
}

func (f *File) WithPath(path string) *File {
	f.Path = filepath.Clean(path)

	return f
}

func (f *File) WithOwnerID(ownerID uuid.UUID) *File {
	f.OwnerID = ownerID

	return f
}

func (f *File) FullPath() string {
	return filepath.Join(f.Path, f.Name)
}

func (f *File) Response() *File {
	f.ShownPath = app.RemoveRootPath(f.Path)

	return f
}

func (f *File) Parents() []string {
	if f.Path == "" || f.Path == "/" {
		return nil
	}

	// Initialize an empty slice to store parent paths
	var result []string

	currentPath := f.Path

	for currentPath != "" && currentPath != "/" {
		result = append(result, currentPath)

		currentPath = filepath.Dir(currentPath)
	}

	return result
}

type SimpleFile struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Path string    `json:"path"`
} // @name file.SimpleFile

func (f *SimpleFile) FullPath() string {
	return filepath.Join(f.Path, f.Name)
}

type Share struct {
	FileID    uuid.UUID `json:"file_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
} // @name file.Share

type Stars struct {
	FileID    uuid.UUID `json:"file_id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
} // @name file.Stars

type Filter struct {
	Type  string
	After *time.Time
}

func NewFilter(_type string, after *time.Time) Filter {
	return Filter{
		Type:  _type,
		After: after,
	}
}

type Log struct {
	FileID    uuid.UUID `json:"file_id"`
	UserID    uuid.UUID `json:"user_id"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`

	File *File          `json:"file,omitempty"`
	User *identity.User `json:"user,omitempty"`
} // @name file.Log

func NewLog(fileID, userID uuid.UUID, action string) Log {
	return Log{
		FileID: fileID,
		UserID: userID,
		Action: action,
	}
}

var (
	LogActionOpen    = "open"
	LogActionCreate  = "create"
	LogActionUpdate  = "update"
	LogActionDelete  = "delete"
	LogActionMove    = "move"
	LogActionShare   = "share"
	LogActionStar    = "star"
	SuggestedActions = []string{LogActionOpen, LogActionCreate, LogActionUpdate, LogActionDelete}
)

type Storage struct {
	Text     uint64 `json:"text"`
	Document uint64 `json:"document"`
	PDF      uint64 `json:"pdf"`
	JSON     uint64 `json:"json"`
	Image    uint64 `json:"image"`
	Video    uint64 `json:"video"`
	Audio    uint64 `json:"audio"`
	Archive  uint64 `json:"archive"`
	Other    uint64 `json:"other"`
}

func NewStorage(files []File) Storage {
	var storage Storage

	for _, file := range files {
		switch file.Type {
		case "text":
			storage.Text += file.Size
		case "document":
			storage.Document += file.Size
		case "pdf":
			storage.PDF += file.Size
		case "json":
			storage.JSON += file.Size
		case "image":
			storage.Image += file.Size
		case "video":
			storage.Video += file.Size
		case "audio":
			storage.Audio += file.Size
		case "archive":
			storage.Archive += file.Size
		default:
			storage.Other += file.Size
		}
	}

	return storage
}

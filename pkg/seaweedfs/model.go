package seaweedfs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrNotFound = errors.New("no such file or directory")
)

type FullPath string

func (fp FullPath) Name() string {
	_, name := filepath.Split(string(fp))

	return strings.ToValidUTF8(name, "?")
}

// split, but skipping the root
func (fp FullPath) Split() []string {
	if fp == "" || fp == "/" {
		return []string{}
	}

	return strings.Split(string(fp)[1:], "/")
}

type Entry struct {
	FullPath      FullPath
	Mtime         time.Time   // time of last modification
	Crtime        time.Time   // time of creation (OS X only)
	Mode          os.FileMode // file mode
	Uid           uint32      // owner uid
	Gid           uint32      // group gid
	Mime          string      // mime type
	TtlSec        int32       // ttl in seconds
	UserName      string
	GroupNames    []string
	SymlinkTarget string
	Md5           []byte
	FileSize      uint64
	Rdev          uint32
	Inode         uint64
}

func (e Entry) IsDirectory() bool {
	return e.Mode&os.ModeDir > 0
}

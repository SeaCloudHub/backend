package app

import (
	"fmt"
	"path/filepath"
	"strings"
)

func GetIdentityDirPath(identityID string) string {
	return fmt.Sprintf("/%s/", identityID)
}

func GetFullRoot(path string, id string) string {
	return filepath.Join(string(filepath.Separator), id, path) + string(filepath.Separator)
}

func GetRootPath(path string) string {
	entryPath := strings.Split(path, "/")
	if len(entryPath) < 3 {
		return "/"
	}

	return "/" + entryPath[1] + "/"
}

// remove the root path from the full path
func RemoveRootPath(fp string) string {
	entryPath := []string{}
	if fp != "" && fp != "/" {
		entryPath = strings.Split(fp[1:], "/")
	}

	if len(entryPath) == 0 {
		return "/"
	}

	entryPath[0] = "/"

	return filepath.Join(entryPath...)
}

// GetPathAndName returns the parent path and name of a file or directory
// fmt.Println(GetPathAndName("/a/b/c.txt"))
// fmt.Println(GetPathAndName("/a/b/c"))
// fmt.Println(GetPathAndName("/a/b/c/"))
// fmt.Println(GetPathAndName("a/b/c/"))
// fmt.Println(GetPathAndName("/a/b/"))
// fmt.Println(GetPathAndName("/a/"))
// fmt.Println(GetPathAndName("/a"))
// /a/b/ c.txt
// /a/b/ c
// /a/b/ c
// a/b/ c
// /a/ b
// / a
// / a
func GetPathAndName(fullPath string) (string, string) {
	if fullPath == "/" {
		return "", "/"
	}

	dir, file := filepath.Split(fullPath)
	if strings.HasSuffix(fullPath, "/") {
		dir, file = filepath.Dir(fullPath[:len(fullPath)-1])+"/", filepath.Base(fullPath[:len(fullPath)-1])
	}

	if dir == "/" || dir == "//" {
		return "/", file
	}

	return dir, file
}

func GetParentPath(fullPath string) string {
	if fullPath == "/" || fullPath == "" {
		return ""
	}

	parent := filepath.Join(fullPath, "..")
	if strings.HasSuffix(parent, "/") {
		return parent
	}

	return parent + string(filepath.Separator)
}

func IsDirPath(path string) bool {
	return strings.HasSuffix(path, "/")
}

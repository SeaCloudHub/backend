package util

import (
	"bufio"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

func BindMultipartFile(c echo.Context, key string) (*bufio.Reader, string, error) {
	var (
		buf         *bufio.Reader
		contentType string
	)

	reader, err := c.Request().MultipartReader()
	if err != nil {
		return nil, "", fmt.Errorf("parsing multipart form: %w", err)
	}

	for {
		part, err := reader.NextPart()
		if err != nil {
			return nil, "", fmt.Errorf("reading multipart form: %w", err)
		}

		if part.FormName() != key {
			continue
		}

		buf = bufio.NewReader(part)

		data, _ := buf.Peek(512)
		contentType = http.DetectContentType(data)

		break
	}

	return buf, contentType, nil
}

func GetIdentityDirPath(identityID string) string {
	return fmt.Sprintf("/%s/", identityID)
}

func GetFullRoot(path string, id string) string {
	return filepath.Join(string(filepath.Separator), id, path) + string(filepath.Separator)
}

// remove the root path from the full path
func RemoveRootPath(fp string) string {
	entryPath := []string{}
	if fp != "" && fp != "/" {
		entryPath = strings.Split(fp[1:], "/")
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

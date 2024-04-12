package app

import (
	"bufio"
	"fmt"
	"net/http"

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

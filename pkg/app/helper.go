package app

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gabriel-vasile/mimetype"
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

// DetectContentType returns the MIME type of input and a new reader
// containing the whole data from input.
func DetectContentType(input io.Reader) (string, io.Reader, error) {
	// header will store the bytes mimetype uses for detection.
	header := bytes.NewBuffer(nil)

	// After DetectReader, the data read from input is copied into header.
	mtype, err := mimetype.DetectReader(io.TeeReader(input, header))
	if err != nil {
		return "", nil, err
	}

	// Concatenate back the header to the rest of the file.
	// recycled now contains the complete, original data.
	recycled := io.MultiReader(header, input)

	return mtype.String(), recycled, err
}

package file

import (
	"io"
	"mime/multipart"
)

type Service interface {
	GetFile(filename string) (io.ReadCloser, error)
	UploadFile(file *multipart.FileHeader) error
}

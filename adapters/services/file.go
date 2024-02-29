package services

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/SeaCloudHub/backend/pkg/config"
)

type FileService struct {
	FilerServer string

	client *http.Client
}

func NewFileService(cfg *config.Config) *FileService {
	return &FileService{
		FilerServer: cfg.SeaweedFS.FilerServer,
		client:      http.DefaultClient,
	}
}

func (s *FileService) GetFile(filename string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", s.FilerServer+"/"+filename, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (s *FileService) UploadFile(file *multipart.FileHeader) error {
	var (
		buf = new(bytes.Buffer)
		w   = multipart.NewWriter(buf)
	)

	f, err := file.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	part, err := w.CreateFormFile("file", file.Filename)
	if err != nil {
		return err
	}

	if _, err := io.Copy(part, f); err != nil {
		return err
	}

	w.Close()

	req, err := http.NewRequest("POST", s.FilerServer+"/upload/", buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

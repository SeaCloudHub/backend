package model

type GetImageRequest struct {
	Name string `param:"name"`
}

type UploadImageResponse struct {
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
} // @name model.UploadImageResponse

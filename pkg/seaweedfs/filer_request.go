package seaweedfs

import "io"

type GetMetadataRequest struct {
	FullPath string
}

type ListEntriesRequest struct {
	DirPath            string
	Limit              int
	LastFileName       string
	NamePattern        string
	NamePatternExclude string
}

type DownloadFileRequest struct {
	FullPath string
}

type UploadFileRequest struct {
	Content      io.Reader
	FullFileName string
}

type CreateDirectoryRequest struct {
	DirPath string
}

type DeleteRequest struct {
	FullPath string
}

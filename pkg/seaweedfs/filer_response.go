package seaweedfs

type ListEntriesResponse struct {
	Path                  string
	Entries               []Entry
	Limit                 int
	LastFileName          string
	ShouldDisplayLoadMore bool
	EmptyFolder           bool
}

type UploadFileResponse struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type AppendFileResponse struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

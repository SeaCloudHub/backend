package seaweedfs

type ListEntriesResponse struct {
	Path                  string
	Entries               []Entry
	Limit                 int
	LastFileName          string
	ShouldDisplayLoadMore bool
	EmptyFolder           bool
}

func (r *ListEntriesResponse) GetTotalSize() uint64 {
	var totalSize uint64

	for _, entry := range r.Entries {
		totalSize += entry.FileSize
	}

	return totalSize
}

type UploadFileResponse struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

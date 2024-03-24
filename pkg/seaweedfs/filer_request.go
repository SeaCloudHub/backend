package seaweedfs

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

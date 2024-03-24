package seaweedfs

type ListEntriesResponse struct {
	Path                  string
	Entries               []Entry
	Limit                 int
	LastFileName          string
	ShouldDisplayLoadMore bool
	EmptyFolder           bool
}

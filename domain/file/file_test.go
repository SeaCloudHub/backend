package file_test

import (
	"testing"

	"github.com/SeaCloudHub/backend/domain/file"
)

func TestParents(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"/r/a/b/c", []string{"/r/a/b/c", "/r/a/b", "/r/a", "/r"}},
		{"/r/a/b", []string{"/r/a/b", "/r/a", "/r"}},
		{"/r/a", []string{"/r/a", "/r"}},
		{"/r", []string{"/r"}},
		{"/", nil},
		{"", nil},
	}

	for _, tt := range tests {
		var f file.File = file.File{Path: tt.path}
		got := f.Parents()
		if len(got) != len(tt.want) {
			t.Errorf("Parents(%q) = %v; want %v", tt.path, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("Parents(%q) = %v; want %v", tt.path, got, tt.want)
				break
			}
		}
	}
}

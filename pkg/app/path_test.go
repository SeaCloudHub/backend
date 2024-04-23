package app

import "testing"

func TestGetRootPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/a/b/c/", "/a/"},
		{"/a/b/", "/a/"},
		{"/a/", "/a/"},
		{"/", "/"},
	}
	for _, tt := range tests {
		got := GetRootPath(tt.path)
		if got != tt.want {
			t.Errorf("GetRootPath(%q) = %q; want %q", tt.path, got, tt.want)
		}
	}
}

package config

import "testing"

func TestLibraryDirPathIsFixed(t *testing.T) {
	t.Setenv("LIBRARY_DIR", "/tmp/should-not-apply")

	if LibraryDirPath != "/nhentai-popular" {
		t.Fatalf("expected fixed library dir path, got %q", LibraryDirPath)
	}
}

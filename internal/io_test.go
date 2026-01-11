package internal

import "testing"

func TestSearchPaths(t *testing.T) {
	t.Setenv("JF_PATH", "/tmp/p1:/tmp/p2")
	searchPaths := SearchPaths()
	if len(searchPaths) != 2 {
		t.Fatalf("expected 2 search paths, got %d", len(searchPaths))
	}
	if searchPaths[0] != "/tmp/p1" {
		t.Fatalf("expected /tmp/p1, got %s", searchPaths[0])
	}
	if searchPaths[1] != "/tmp/p2" {
		t.Fatalf("expected /tmp/p2, got %s", searchPaths[1])
	}
}

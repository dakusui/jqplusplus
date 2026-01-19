package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func WriteTempJSON(t *testing.T, dir, name string, data string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(data), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

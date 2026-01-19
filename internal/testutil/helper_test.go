package testutil

import (
	"os"
	"testing"
)

func TestWriteTempJSON(t *testing.T) {
	dir := t.TempDir()
	data := `def custom_func:
  { new_key: .store };
`
	path := WriteTempJSON(t, dir, "parent.jq",
		data)

	// Verify the file content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}
	if string(content) != data {
		t.Errorf("file content does not match expected data")
	}
}

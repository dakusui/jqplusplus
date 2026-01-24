package main

import (
	"encoding/json"
	"fmt"
	"github.com/dakusui/jqplusplus/internal"
	"github.com/dakusui/jqplusplus/internal/testutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProcessNodeEntry(t *testing.T) {
	fmt.Sprintf("Hello")
}

func TestLoadAndResolveInheritances_SingleExtendsForJqFile(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.jq",
		`def custom_func:
  { new_key: .store };
`)
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
  "$extends": ["parent.jq"],
  "store": "Hello",
  "key": "eval:object:parent::custom_func"
}`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected, _ := json.MarshalIndent(map[string]any{"key": map[string]any{"new_key": "Hello"}, "store": "Hello"}, "", "  ")
	if !reflect.DeepEqual(result, string(expected)) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}
func TestLoadAndResolveInheritances_TagRef(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
  "thetag": "Hello, mytag",
  "k0": {
    "k1": "eval:string:reftag(\"thetag\")"
  }
}`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected, _ := json.MarshalIndent(
		map[string]any{
			"thetag": "Hello, mytag",
			"k0": map[string]any{
				"k1": "Hello, mytag"}},
		"", "  ")
	if !reflect.DeepEqual(result, string(expected)) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

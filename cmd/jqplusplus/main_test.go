package main

import (
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
  { new_key: . };
`)
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
  "$extends": ["parent.jq"],
  "key": "eval:custom_func"
}`)
	result, err := processNodeEntryKey(internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child)))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(3), "c": float64(4)}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

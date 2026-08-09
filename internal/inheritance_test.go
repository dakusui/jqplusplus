package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dakusui/jqplusplus/internal/testutil"
)

func TestLoadAndResolveInheritances_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := testutil.WriteTempJSON(t, dir, "base.json", `{"a": 1, "b": 2}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(file), filepath.Base(file), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(2)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_SingleExtends(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": ["parent.json"], "b": 3, "c": 4}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(3), "c": float64(4)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_OptionalExtends(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "$extends": [
    "O.json?"
  ],
  "hello": "world"
}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"hello": "world"}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_MultipleExtends(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	_ = testutil.WriteTempJSON(t, dir, "p2.json", `{"b": 20, "c": 30}`)
	_ = testutil.WriteTempJSON(t, dir, "p3.json", `{"b": 21}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": ["p1.json", "p2.json", "p3.json"], "c": 300, "d": 400}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(2), "c": float64(300), "d": float64(400)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_SingleInternalExtends(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"x": {"$extends": ["parent.json"], "b": 3, "c": 4}}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"x": map[string]any{"a": float64(1), "b": float64(3), "c": float64(4)}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_SingleLocalExtends(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$local": {"A":{"a": 1, "b": 2}}, "x": {"$extends": ["A"], "b": 3, "c": 4}}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"x": map[string]any{"a": float64(1), "b": float64(3), "c": float64(4)}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_ExtendsLocalNodeInParent(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `
{
  "$local": {
    "X": {
      "x": "valueInX"
    }
  },
  "name": "parent.json"
}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `
{
  "$extends": [
    "parent.json"
  ],
  "i": {
    "$extends": [
      "X"
    ]
  },
  "name": "child.json"
}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{
		"i": map[string]any{
			"x": "valueInX",
		},
		"name": "child.json",
	}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_InternalExtendsLocalNodeInParent(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `
{
  "$local": {
    "X": {
      "x": "valueInX"
    }
  }
}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `
{
  "i": {
    "$extends": [
      "parent.json"
    ],
    "Y": {
      "$extends": [
        "X"
      ]
    }
  }
}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{
		"i": map[string]any{
			"Y": map[string]any{
				"x": "valueInX",
			},
		},
	}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_ExtendsWithLocalOutOfScope_ThenFail(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `
{
  "$local": {
    "X": {
      "x": "valueInX"
    }
  }
}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `
{
  "i": {
    "$extends": [
      "parent.json"
    ]
  },
  "j": {
    "$extends": ["X"]
  }
}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil {
		t.Fatalf("expected error was not raised: %v", result)
	}
}

func TestLoadAndResolveInheritances_SingleExtendsAtFileLevelNonExisting_ThenFail(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": ["nonExisting.json"], "b": 3, "c": 4}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil {
		t.Fatalf("expected error was not raised: %v", result)
	}
	if !strings.Contains(err.Error(), "file not found") || !strings.Contains(err.Error(), "nonExisting") {
		t.Fatalf("unexpected error message: %v", err.Error())
	}
	fmt.Print(err.Error())
}

func TestLoadAndResolveInheritances_SingleExtendsAtNodeLevelNonExisting_ThenFail(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"key":{"$extends": ["nonExisting.json"], "b": 3, "c": 4}}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil {
		t.Fatalf("expected error was not raised: %v", result)
	}
	if !strings.Contains(err.Error(), "file not found") || !strings.Contains(err.Error(), "nonExisting") {
		t.Fatalf("unexpected error message: %v", err.Error())
	}
	fmt.Print(err.Error())
}

func TestLoadAndResolveInheritances_ExtendsRelativelyAndIndirectly(t *testing.T) {
	dir := t.TempDir()
	dir = filepath.Join(dir, "dir")
	os.Mkdir(dir, 0o755)
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `
{
  "$extends": ["Y.json"],
  "x": "X"
}`)
	_ = testutil.WriteTempJSON(t, dir, "Y.json", `
{
  "y": "Y"
}`)
	child := testutil.WriteTempJSON(t, filepath.Dir(dir), "child.json", `
{
  "$extends": [
    "dir/parent.json"
  ],
  "o": "hello world"
}
`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{
		"o": "hello world",
		"x": "X",
		"y": "Y",
	}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_SingleExtendsWithNonString_ThenFail(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": [1234], "b": 3, "c": 4}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil {
		t.Fatalf("expected error was not raised: %v", result)
	}
	if !strings.Contains(err.Error(), "$extends array must contain only strings") || !strings.Contains(err.Error(), "1234") {
		t.Fatalf("unexpected error message: %v", err.Error())
	}
}

func TestLoadAndResolveInheritances_SingleExtendsMalformed_ThenFail(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "malformed.json", `xyz`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": ["malformed.json"], "b": 3, "c": 4}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil {
		t.Fatalf("expected error was not raised: %v", result)
	}
	if !strings.Contains(err.Error(), "invalid character 'x' looking for beginning of value") {
		t.Fatalf("unexpected error message: %v", err.Error())
	}
}

func TestLoadAndResolveInheritances_SingleExtendsWithString_ThenFail(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": "parent.json", "b": 3, "c": 4}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil {
		t.Fatalf("error expected but returned: %v", result)
	}
}

func TestLoadAndResolveInheritances_NestedExtends(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "grand.json", `{"a": 1}`)
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"$extends": ["grand.json"], "b": 2}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": ["parent.json"], "c": 3}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(2), "c": float64(3)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_CircularExtends(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "p1.json", `{"$extends": ["p2.json"]}`)
	_ = testutil.WriteTempJSON(t, dir, "p2.json", `{"$extends": ["p1.json"]}`)
	_, err := LoadAndResolveInheritances(dir, "p1.json", []string{})
	if err == nil || err.Error() == "" {
		t.Errorf("expected error for circular filelevel, got: %v", err)
	}
	fmt.Println(err)
}

func TestLoadAndResolveInheritances_SingleIncludes(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$includes": ["parent.json"], "b": 3, "c": 4}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(2), "c": float64(4)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}
func TestLoadAndResolveInheritances_MultipleIncludes(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	_ = testutil.WriteTempJSON(t, dir, "p2.json", `{"b": 20, "c": 30}`)
	_ = testutil.WriteTempJSON(t, dir, "p3.json", `{"b": 21}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$includes": ["p1.json", "p2.json", "p3.json"], "c": 300, "d": 400}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(21), "c": float64(30), "d": float64(400)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_BothExtendsAndIncludes(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	_ = testutil.WriteTempJSON(t, dir, "p2.json", `{"b": 20, "c": 30}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": ["p1.json"], "$includes": ["p2.json"], "b":21, "c": 300, "d": 400}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(20), "c": float64(30), "d": float64(400)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_BothExtendsAndIncludesTheSame(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": ["p1.json"], "$includes": ["p1.json"], "b":21, "c": 300, "d": 400}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(2), "c": float64(300), "d": float64(400)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_BothExtendsAndIncludesOneLevelNested(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	_ = testutil.WriteTempJSON(t, dir, "p2.json", `{"b": 20, "c": 30}`)
	_ = testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": ["p1.json"], "$includes": ["p2.json"], "b":21, "c": 300, "d": 400}`)
	grandchild := testutil.WriteTempJSON(t, dir, "grandchild.json", `{"$extends": ["child.json"]}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(grandchild), filepath.Base(grandchild), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(20), "c": float64(30), "d": float64(400)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_ExplicitArraySplicePositions(t *testing.T) {
	tests := []struct {
		name  string
		array string
		want  []any
	}{
		{name: "append", array: `["$super", "child"]`, want: []any{"parent", "child"}},
		{name: "prepend", array: `["child", "$super"]`, want: []any{"child", "parent"}},
		{name: "wrap", array: `["before", "$super", "after"]`, want: []any{"before", "parent", "after"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"items":["parent"]}`)
			child := testutil.WriteTempJSON(t, dir, "child.json", fmt.Sprintf(`{"$extends":["parent.json"],"items":%s}`, tt.array))

			result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := result.Obj["items"]
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadAndResolveInheritances_ExplicitArrayPairwiseMerge(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{
  "items": [
    {"name":"old", "keep":true},
    "parent-extra"
  ]
}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "$extends": ["parent.json"],
  "items": ["prefix", "$super*", {"name":"new"}]
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{
		"prefix",
		map[string]any{"name": "new", "keep": true},
		"parent-extra",
	}
	if !reflect.DeepEqual(result.Obj["items"], want) {
		t.Fatalf("got %v, want %v", result.Obj["items"], want)
	}
}

func TestLoadAndResolveInheritances_ArrayDeltasComposeThroughIncludes(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "base.json", `{"tags":["base"]}`)
	_ = testutil.WriteTempJSON(t, dir, "mixin.json", `{"tags":["$super","mixin"]}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "$extends":["base.json"],
  "$includes":["mixin.json"]
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{"base", "mixin"}
	if !reflect.DeepEqual(result.Obj["tags"], want) {
		t.Fatalf("got %v, want %v", result.Obj["tags"], want)
	}
}

func TestLoadAndResolveInheritances_MultipleSpliceMixinsCompose(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "base.json", `{"tags":["base"]}`)
	_ = testutil.WriteTempJSON(t, dir, "first.json", `{"tags":["$super","first"]}`)
	_ = testutil.WriteTempJSON(t, dir, "second.json", `{"tags":["$super","second"]}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "$extends":["base.json"],
  "$includes":["first.json","second.json"]
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{"base", "first", "second"}
	if !reflect.DeepEqual(result.Obj["tags"], want) {
		t.Fatalf("got %v, want %v", result.Obj["tags"], want)
	}
}

func TestLoadAndResolveInheritances_ArrayDeltasComposeAcrossLevels(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "base.json", `{"tags":["base"]}`)
	_ = testutil.WriteTempJSON(t, dir, "middle.json", `{
  "$extends":["base.json"],
  "tags":["$super","middle"]
}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "$extends":["middle.json"],
  "tags":["$super","child"]
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{"base", "middle", "child"}
	if !reflect.DeepEqual(result.Obj["tags"], want) {
		t.Fatalf("got %v, want %v", result.Obj["tags"], want)
	}
}

func TestLoadAndResolveInheritances_DanglingArrayMarkerIsAnError(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"items":["$super","child"]}`)

	_, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil || !strings.Contains(err.Error(), "dangling array merge token") {
		t.Fatalf("error = %v, want dangling array merge token", err)
	}
}

func TestLoadAndResolveInheritances_ReservedArrayMarkerValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr string
	}{
		{name: "unknown optional spelling", value: `["$super?", "child"]`, wantErr: "unknown $super-family token"},
		{name: "unknown function spelling", value: `["$super*(name)"]`, wantErr: "unknown $super-family token"},
		{name: "identifier literal", value: `["$supervisor"]`, wantErr: ""},
		{name: "raw escape", value: `["raw:$super"]`, wantErr: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			child := testutil.WriteTempJSON(t, dir, "child.json", fmt.Sprintf(`{"items":%s}`, tt.value))
			_, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadAndResolveInheritances_MarkedArrayRequiresArrayParent(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"items":{"name":"parent"}}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends":["parent.json"],"items":["$super","child"]}`)

	_, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil || !strings.Contains(err.Error(), "explicit array merge requires an inherited array") {
		t.Fatalf("error = %v, want inherited-array mismatch", err)
	}
}

func TestLoadAndResolveInheritances_DoubleMarkedArraysAreAnError(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"items":["$super*",{"name":"parent"}]}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends":["parent.json"],"items":["$super*",{"name":"child"}]}`)

	_, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil || !strings.Contains(err.Error(), "marked arrays cannot be merged") {
		t.Fatalf("error = %v, want double-marked array error", err)
	}
}

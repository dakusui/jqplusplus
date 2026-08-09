package internal

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dakusui/jqplusplus/internal/testutil"
)

func TestArrayComposition_Splice(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"items": ["parent", {"source": "parent"}]}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "$extends": ["parent.json"],
  "items": ["before", "$super", "after", "$super"]
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{
		"items": []any{"before", "parent", map[string]any{"source": "parent"}, "after", "parent", map[string]any{"source": "parent"}},
	}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %#v, got %#v", expected, result.Obj)
	}
}

func TestArrayComposition_SpliceDeltaSurvivesIncludes(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "mixin.json", `{"tags": ["$super", "mandated"]}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "$includes": ["mixin.json"],
  "tags": ["base"]
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"tags": []any{"base", "mandated"}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %#v, got %#v", expected, result.Obj)
	}
}

func TestArrayComposition_SpliceDeltaComposesAcrossInheritance(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "base.json", `{"items": ["base"]}`)
	_ = testutil.WriteTempJSON(t, dir, "middle.json", `{"$extends": ["base.json"], "items": ["$super", "middle"]}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": ["middle.json"], "items": ["$super", "child"]}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"items": []any{"base", "middle", "child"}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %#v, got %#v", expected, result.Obj)
	}
}

func TestArrayComposition_NodeLevelSplice(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"items": ["parent"]}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "section": {
    "$extends": ["parent.json"],
    "items": ["child", "$super"]
  }
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"section": map[string]any{"items": []any{"child", "parent"}}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %#v, got %#v", expected, result.Obj)
	}
}

func TestArrayComposition_Pairwise(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{
  "items": [{"a": 1, "from_parent": true}, {"b": 2}]
}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "$extends": ["parent.json"],
  "items": ["prefix", "$super*", {"a": 9}, {"extra": 4}, "tail"]
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"items": []any{
		"prefix",
		map[string]any{"a": float64(9), "from_parent": true},
		map[string]any{"b": float64(2), "extra": float64(4)},
		"tail",
	}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %#v, got %#v", expected, result.Obj)
	}
}

func TestArrayComposition_PairwiseKeepsParentExtras(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"items": [{"a": 1}, {"b": 2}, {"c": 3}]}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "$extends": ["parent.json"],
  "items": ["prefix", "$super*", {"a": 9}, {"extra": 4}]
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"items": []any{
		"prefix",
		map[string]any{"a": float64(9)},
		map[string]any{"b": float64(2), "extra": float64(4)},
		map[string]any{"c": float64(3)},
	}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %#v, got %#v", expected, result.Obj)
	}
}

func TestArrayComposition_PairwiseRecursesAndPadsObjects(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"items": [[{"kept": true}]]}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "$extends": ["parent.json"],
  "items": ["$super*", ["$super*", {}]]
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"items": []any{[]any{map[string]any{"kept": true}}}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %#v, got %#v", expected, result.Obj)
	}
}

func TestArrayComposition_PairwiseAtomsUseChildValues(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json", `{"items": [1, true, null]}`)
	child := testutil.WriteTempJSON(t, dir, "child.json", `{"$extends": ["parent.json"], "items": ["$super*", 2, false, "child"]}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"items": []any{float64(2), false, "child"}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %#v, got %#v", expected, result.Obj)
	}
}

func TestArrayComposition_ValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		parent   string
		child    string
		contains string
	}{
		{
			name:     "unresolved splice",
			child:    `{"items": ["$super"]}`,
			contains: "unresolved array composition token \"$super\"",
		},
		{
			name:     "unresolved pairwise",
			child:    `{"items": ["$super*"]}`,
			contains: "unresolved array composition token \"$super*\"",
		},
		{
			name:     "unknown reserved token",
			child:    `{"items": ["$super?"]}`,
			contains: "unknown array composition token \"$super?\"",
		},
		{
			name:     "mixed tokens",
			child:    `{"items": ["$super", "$super*"]}`,
			contains: "cannot mix $super and $super*",
		},
		{
			name:     "duplicate pairwise token",
			child:    `{"items": ["$super*", "$super*"]}`,
			contains: "only one $super*",
		},
		{
			name:     "non array parent",
			parent:   `{"items": "parent"}`,
			child:    `{"$extends": ["parent.json"], "items": ["$super"]}`,
			contains: "$super requires an inherited array",
		},
		{
			name:     "cross kind pair",
			parent:   `{"items": [{"parent": true}]}`,
			child:    `{"$extends": ["parent.json"], "items": ["$super*", "child"]}`,
			contains: "cannot pairwise merge object with atom",
		},
		{
			name:     "two marked arrays",
			parent:   `{"items": ["$super", "parent"]}`,
			child:    `{"$extends": ["parent.json"], "items": ["$super", "child"]}`,
			contains: "cannot merge two marked arrays",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if test.parent != "" {
				_ = testutil.WriteTempJSON(t, dir, "parent.json", test.parent)
			}
			child := testutil.WriteTempJSON(t, dir, "child.json", test.child)
			_, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), nil)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), test.contains) {
				t.Errorf("expected error containing %q, got %q", test.contains, err)
			}
		})
	}
}

func TestArrayComposition_RawEscapeAndIdentifierLiterals(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json", `{
  "items": ["raw:$super", "$supervisor", "$superuser_home"]
}`)

	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"items": []any{"raw:$super", "$supervisor", "$superuser_home"}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %#v, got %#v", expected, result.Obj)
	}
}

package internal

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dakusui/jqplusplus/internal/testutil"
)

func TestLoadAndResolveInheritances_ComposesArraysAtFileAndNodeLevel(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteTempJSON(t, dir, "parent.json", `{
  "append": ["parent"],
  "pair": [{"inherited": true}, {"kept": true}],
  "replace": ["parent"]
}`)
	input := testutil.WriteTempJSON(t, dir, "input.json", `{
  "$extends": ["parent.json"],
  "append": ["$super", "child"],
  "pair": ["prefix", "$super*", {"child": true}, {}, "tail"],
  "replace": ["child"],
  "node": {
    "$extends": ["parent.json"],
    "append": ["node", "$super"]
  }
}`)

	got, err := LoadAndResolveInheritances(filepath.Dir(input), filepath.Base(input), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{
		"append":  []any{"parent", "child"},
		"pair":    []any{"prefix", map[string]any{"inherited": true, "child": true}, map[string]any{"kept": true}, "tail"},
		"replace": []any{"child"},
		"node": map[string]any{
			"append":  []any{"node", "parent"},
			"pair":    []any{map[string]any{"inherited": true}, map[string]any{"kept": true}},
			"replace": []any{"parent"},
		},
	}
	if !reflect.DeepEqual(got.Obj, want) {
		t.Fatalf("got %v, want %v", got.Obj, want)
	}
}

func TestLoadAndResolveInheritances_IncludesCanCarrySuperDelta(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteTempJSON(t, dir, "mixin.json", `{"tags": ["$super", "mandated"]}`)
	input := testutil.WriteTempJSON(t, dir, "input.json", `{
  "$includes": ["mixin.json"],
  "tags": ["local"]
}`)

	got, err := LoadAndResolveInheritances(filepath.Dir(input), filepath.Base(input), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"tags": []any{"local", "mandated"}}
	if !reflect.DeepEqual(got.Obj, want) {
		t.Fatalf("got %v, want %v", got.Obj, want)
	}
}

func TestLoadAndResolveInheritances_ComposesSuperDeltasAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteTempJSON(t, dir, "base.json", `{"tags": ["base"]}`)
	testutil.WriteTempJSON(t, dir, "first.json", `{"tags": ["first", "$super"]}`)
	testutil.WriteTempJSON(t, dir, "second.json", `{
  "$extends": ["first.json"],
  "tags": ["$super", "second"]
}`)
	input := testutil.WriteTempJSON(t, dir, "input.json", `{
  "$extends": ["second.json", "base.json"]
}`)

	got, err := LoadAndResolveInheritances(filepath.Dir(input), filepath.Base(input), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"tags": []any{"first", "base", "second"}}
	if !reflect.DeepEqual(got.Obj, want) {
		t.Fatalf("got %v, want %v", got.Obj, want)
	}
}

func TestLoadAndResolveInheritances_ReportsArrayCompositionErrors(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		input  string
		want   string
	}{
		{
			name:  "dangling token",
			input: `{"missing": ["$super", "child"]}`,
			want:  "unresolved array-composition token",
		},
		{
			name:   "non-array inherited value",
			parent: `{"values": "atom"}`,
			input:  `{"$extends": ["parent.json"], "values": ["$super"]}`,
			want:   "expected array",
		},
		{
			name:  "unknown token family member",
			input: `{"values": ["$super?"]}`,
			want:  "unknown array-composition token",
		},
		{
			name:   "mixed tokens",
			parent: `{"values": []}`,
			input:  `{"$extends": ["parent.json"], "values": ["$super", "$super*"]}`,
			want:   "cannot be used in the same array",
		},
		{
			name:   "two pairwise deltas",
			parent: `{"values": ["$super*"]}`,
			input:  `{"$extends": ["parent.json"], "values": ["$super*"]}`,
			want:   "cannot compose marked arrays",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.parent != "" {
				testutil.WriteTempJSON(t, dir, "parent.json", tt.parent)
			}
			input := testutil.WriteTempJSON(t, dir, "input.json", tt.input)
			_, err := LoadAndResolveInheritances(filepath.Dir(input), filepath.Base(input), nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got error %v, want substring %q", err, tt.want)
			}
		})
	}
}

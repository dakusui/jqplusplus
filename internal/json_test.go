package internal

import (
	"github.com/dakusui/jqplusplus/internal/testutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadAndResolveInheritancesWithYaml_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := testutil.WriteTempJSON(t, dir, "base.yaml", `
a: 1
b: 2
`)
	result, err := LoadAndResolveInheritances(filepath.Dir(file), filepath.Base(file), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": int(1), "b": int(2)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritancesWithToml_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := testutil.WriteTempJSON(t, dir, "base.toml", `
	a = 1
	b = 2
	`)
	result, err := LoadAndResolveInheritances(filepath.Dir(file), filepath.Base(file), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": int64(1), "b": int64(2)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritancesWithJSON5_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := testutil.WriteTempJSON(t, dir, "base.json5", `
	{
	  a: 1,
	  b: 2,
	}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(file), filepath.Base(file), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(2)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritancesWithHOCON_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := testutil.WriteTempJSON(t, dir, "base.hocon", `
a = 1
b = 2
`)
	result, err := LoadAndResolveInheritances(filepath.Dir(file), filepath.Base(file), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": int(1), "b": int(2)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeObjects_DeepMerge(t *testing.T) {
	parent := map[string]interface{}{
		"a": 1,
		"b": map[string]interface{}{
			"x": 10,
			"y": 20,
		},
	}
	child := map[string]interface{}{
		"b": map[string]interface{}{
			"y": 200,
			"z": 300,
		},
		"c": 3,
	}
	expected := map[string]interface{}{
		"a": 1,
		"b": map[string]interface{}{
			"x": 10,
			"y": 200,
			"z": 300,
		},
		"c": 3,
	}
	result := mergeObjects(parent, child)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeObjects(t *testing.T) {
	t.Run("shallow merge", func(t *testing.T) {
		a := map[string]interface{}{"a": 1, "b": 2}
		b := map[string]interface{}{"b": 3, "c": 4}
		expected := map[string]interface{}{"a": 1, "b": 3, "c": 4}
		result := MergeObjects(a, b, MergePolicyDefault)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("deep merge", func(t *testing.T) {
		a := map[string]interface{}{
			"a": 1,
			"b": map[string]interface{}{"x": 10, "y": 20},
		}
		b := map[string]interface{}{
			"b": map[string]interface{}{"y": 200, "z": 300},
			"c": 3,
		}
		expected := map[string]interface{}{
			"a": 1,
			"b": map[string]interface{}{"x": 10, "y": 200, "z": 300},
			"c": 3,
		}
		result := MergeObjects(a, b, MergePolicyDefault)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("primitive overwrite", func(t *testing.T) {
		a := map[string]interface{}{"a": 1, "b": 2}
		b := map[string]interface{}{"b": 100}
		expected := map[string]interface{}{"a": 1, "b": 100}
		result := MergeObjects(a, b, MergePolicyDefault)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("empty maps", func(t *testing.T) {
		a := map[string]interface{}{}
		b := map[string]interface{}{"a": 1}
		expected := map[string]interface{}{"a": 1}
		result := MergeObjects(a, b, MergePolicyDefault)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}

		a2 := map[string]interface{}{"a": 1}
		b2 := map[string]interface{}{}
		expected2 := map[string]interface{}{"a": 1}
		result2 := MergeObjects(a2, b2, MergePolicyDefault)
		if !reflect.DeepEqual(result2, expected2) {
			t.Errorf("expected %v, got %v", expected2, result2)
		}
	})
}

func TestMergeObjects_DefaultPolicyReplacesArrays(t *testing.T) {
	parent := map[string]any{"items": []any{"parent"}}
	child := map[string]any{"items": []any{"child"}}

	got := MergeObjects(parent, child, MergePolicyDefault)
	want := map[string]any{"items": []any{"child"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default policy unexpectedly composed arrays: got %v, want %v", got, want)
	}
}

func TestMergeObjects_InheritanceSplicesAtEveryMarker(t *testing.T) {
	parent := map[string]any{"items": []any{"p0", map[string]any{"name": "p1"}}}
	child := map[string]any{"items": []any{"before", "$super", "after", "$super"}}

	got, err := mergeObjectsWithError(parent, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"items": []any{
		"before",
		"p0",
		map[string]any{"name": "p1"},
		"after",
		"p0",
		map[string]any{"name": "p1"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	gotItems := got["items"].([]any)
	gotItems[1] = "changed"
	if parent["items"].([]any)[0] != "p0" {
		t.Fatal("splice result aliases the inherited array")
	}
}

func TestMergeObjects_InheritanceSpliceDeltasCompose(t *testing.T) {
	parent := map[string]any{"items": []any{"$super", "parent"}}
	child := map[string]any{"items": []any{"$super", "child"}}

	got, err := mergeObjectsWithError(parent, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"items": []any{"$super", "parent", "child"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeObjects_InheritancePairwiseMerge(t *testing.T) {
	parent := map[string]any{"items": []any{
		map[string]any{"name": "old", "keep": true},
		"parent-extra",
	}}
	child := map[string]any{"items": []any{
		"prefix",
		"$super*",
		map[string]any{"name": "new"},
	}}

	got, err := mergeObjectsWithError(parent, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"items": []any{
		"prefix",
		map[string]any{"name": "new", "keep": true},
		"parent-extra",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeObjects_InheritancePairwiseKeepsChildTail(t *testing.T) {
	parent := map[string]any{"items": []any{
		map[string]any{"name": "old"},
	}}
	child := map[string]any{"items": []any{
		"$super*",
		map[string]any{"name": "new"},
		"child-tail",
	}}

	got, err := mergeObjectsWithError(parent, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"items": []any{
		map[string]any{"name": "new"},
		"child-tail",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeObjects_InheritancePairwiseEmptyObjectKeepsParent(t *testing.T) {
	parent := map[string]any{"items": []any{
		map[string]any{"name": "old", "keep": true},
	}}
	child := map[string]any{"items": []any{"$super*", map[string]any{}}}

	got, err := mergeObjectsWithError(parent, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"items": []any{
		map[string]any{"name": "old", "keep": true},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeObjects_InheritancePairwiseMergesNestedMarkedArrays(t *testing.T) {
	parent := map[string]any{"items": []any{
		map[string]any{"tags": []any{"p0", "p1"}},
	}}
	child := map[string]any{"items": []any{
		"$super*",
		map[string]any{"tags": []any{"$super", "c"}},
	}}

	got, err := mergeObjectsWithError(parent, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"items": []any{
		map[string]any{"tags": []any{"p0", "p1", "c"}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeObjects_InheritanceRejectsInvalidPairs(t *testing.T) {
	tests := []struct {
		name   string
		parent any
		child  any
		want   string
	}{
		{name: "array and object", parent: []any{"parent"}, child: []any{"$super*", map[string]any{"x": 1}}, want: "array pair at index 0: cannot merge atom with object"},
		{name: "object and array", parent: map[string]any{"x": 1}, child: []any{"$super", "child"}, want: "explicit array merge requires an inherited array"},
		{name: "atom and array", parent: "parent", child: []any{"$super", "child"}, want: "explicit array merge requires an inherited array"},
		{name: "mixed markers", parent: []any{"parent"}, child: []any{"$super", "$super*"}, want: "array cannot mix"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mergeObjectsWithError(map[string]any{"value": tt.parent}, map[string]any{"value": tt.child})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

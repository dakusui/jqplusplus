package inheritance

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeTempJSON(t *testing.T, dir, name string, data string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(data), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func TestLoadAndResolve_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := writeTempJSON(t, dir, "base.json", `{"a": 1, "b": 2}`)
	result, err := LoadAndResolve(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(2)}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolve_SingleExtends(t *testing.T) {
	dir := t.TempDir()
	_ = writeTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": ["parent.json"], "b": 3, "c": 4}`)
	result, err := LoadAndResolve(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(3), "c": float64(4)}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolve_SingleExtendsWithString_ThenFail(t *testing.T) {
	dir := t.TempDir()
	_ = writeTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": "parent.json", "b": 3, "c": 4}`)
	result, err := LoadAndResolve(child)
	if err == nil {
		t.Fatalf("error expected but returned: %v", result)
	}
}

func TestLoadAndResolve_MultipleExtends(t *testing.T) {
	dir := t.TempDir()
	_ = writeTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	_ = writeTempJSON(t, dir, "p2.json", `{"b": 20, "c": 30}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": ["p1.json", "p2.json"], "c": 300, "d": 400}`)
	result, err := LoadAndResolve(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(2), "c": float64(300), "d": float64(400)}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolve_NestedExtends(t *testing.T) {
	dir := t.TempDir()
	_ = writeTempJSON(t, dir, "grand.json", `{"a": 1}`)
	_ = writeTempJSON(t, dir, "parent.json", `{"$extends": ["grand.json"], "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": ["parent.json"], "c": 3}`)
	result, err := LoadAndResolve(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(2), "c": float64(3)}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolve_CircularExtends(t *testing.T) {
	dir := t.TempDir()
	_ = writeTempJSON(t, dir, "p1.json", `{"$extends": ["p2.json"]}`)
	_ = writeTempJSON(t, dir, "p2.json", `{"$extends": ["p1.json"]}`)
	_, err := LoadAndResolve(filepath.Join(dir, "p1.json"))
	if err == nil || err.Error() == "" {
		t.Errorf("expected error for circular inheritance, got: %v", err)
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

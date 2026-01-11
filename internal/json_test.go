package internal

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadAndResolveInheritancesWithYaml_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := writeTempJSON(t, dir, "base.yaml", `
a: 1
b: 2
`)
	result, err := LoadAndResolveInheritances(filepath.Dir(file), filepath.Base(file), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": int(1), "b": int(2)}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritancesWithToml_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := writeTempJSON(t, dir, "base.toml", `
	a = 1
	b = 2
	`)
	result, err := LoadAndResolveInheritances(filepath.Dir(file), filepath.Base(file), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": int64(1), "b": int64(2)}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritancesWithJSON5_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := writeTempJSON(t, dir, "base.json5", `
	{
	  a: 1,
	  b: 2,
	}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(file), filepath.Base(file), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(2)}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritancesWithHOCON_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := writeTempJSON(t, dir, "base.hocon", `
a = 1
b = 2
`)
	result, err := LoadAndResolveInheritances(filepath.Dir(file), filepath.Base(file), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": int(1), "b": int(2)}
	if !reflect.DeepEqual(result, expected) {
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

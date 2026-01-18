package internal

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadAndResolveInheritances_NoExtends(t *testing.T) {
	dir := t.TempDir()
	file := writeTempJSON(t, dir, "base.json", `{"a": 1, "b": 2}`)
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
	_ = writeTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": ["parent.json"], "b": 3, "c": 4}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(3), "c": float64(4)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_MultipleExtends(t *testing.T) {
	dir := t.TempDir()
	_ = writeTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	_ = writeTempJSON(t, dir, "p2.json", `{"b": 20, "c": 30}`)
	_ = writeTempJSON(t, dir, "p3.json", `{"b": 21}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": ["p1.json", "p2.json", "p3.json"], "c": 300, "d": 400}`)
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
	_ = writeTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"x": {"$extends": ["parent.json"], "b": 3, "c": 4}}`)
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
	child := writeTempJSON(t, dir, "child.json", `{"$local": {"A":{"a": 1, "b": 2}}, "x": {"$extends": ["A"], "b": 3, "c": 4}}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]any{"x": map[string]any{"a": float64(1), "b": float64(3), "c": float64(4)}}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLoadAndResolveInheritances_SingleExtendsNonExisting_ThenFail(t *testing.T) {
	dir := t.TempDir()
	_ = writeTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": ["nonExisting.json"], "b": 3, "c": 4}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil {
		t.Fatalf("expected error was not raised: %v", result)
	}
	if !strings.Contains(err.Error(), "file not found") || !strings.Contains(err.Error(), "nonExisting") {
		t.Fatalf("unexpected error message: %v", err.Error())
	}
}

func TestLoadAndResolveInheritances_SingleExtendsWithNonString_ThenFail(t *testing.T) {
	dir := t.TempDir()
	_ = writeTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": [1234], "b": 3, "c": 4}`)
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
	_ = writeTempJSON(t, dir, "malformed.json", `xyz`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": ["malformed.json"], "b": 3, "c": 4}`)
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
	_ = writeTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": "parent.json", "b": 3, "c": 4}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(child), filepath.Base(child), []string{})
	if err == nil {
		t.Fatalf("error expected but returned: %v", result)
	}
}

func TestLoadAndResolveInheritances_NestedExtends(t *testing.T) {
	dir := t.TempDir()
	_ = writeTempJSON(t, dir, "grand.json", `{"a": 1}`)
	_ = writeTempJSON(t, dir, "parent.json", `{"$extends": ["grand.json"], "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": ["parent.json"], "c": 3}`)
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
	_ = writeTempJSON(t, dir, "p1.json", `{"$extends": ["p2.json"]}`)
	_ = writeTempJSON(t, dir, "p2.json", `{"$extends": ["p1.json"]}`)
	_, err := LoadAndResolveInheritances(dir, "p1.json", []string{})
	if err == nil || err.Error() == "" {
		t.Errorf("expected error for circular filelevel, got: %v", err)
	}
}

func TestLoadAndResolveInheritances_SingleIncludes(t *testing.T) {
	dir := t.TempDir()
	_ = writeTempJSON(t, dir, "parent.json", `{"a": 1, "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"$includes": ["parent.json"], "b": 3, "c": 4}`)
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
	_ = writeTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	_ = writeTempJSON(t, dir, "p2.json", `{"b": 20, "c": 30}`)
	_ = writeTempJSON(t, dir, "p3.json", `{"b": 21}`)
	child := writeTempJSON(t, dir, "child.json", `{"$includes": ["p1.json", "p2.json", "p3.json"], "c": 300, "d": 400}`)
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
	_ = writeTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	_ = writeTempJSON(t, dir, "p2.json", `{"b": 20, "c": 30}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": ["p1.json"], "$includes": ["p2.json"], "b":21, "c": 300, "d": 400}`)
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
	_ = writeTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	child := writeTempJSON(t, dir, "child.json", `{"$extends": ["p1.json"], "$includes": ["p1.json"], "b":21, "c": 300, "d": 400}`)
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
	_ = writeTempJSON(t, dir, "p1.json", `{"a": 1, "b": 2}`)
	_ = writeTempJSON(t, dir, "p2.json", `{"b": 20, "c": 30}`)
	_ = writeTempJSON(t, dir, "child.json", `{"$extends": ["p1.json"], "$includes": ["p2.json"], "b":21, "c": 300, "d": 400}`)
	grandchild := writeTempJSON(t, dir, "grandchild.json", `{"$extends": ["child.json"]}`)
	result, err := LoadAndResolveInheritances(filepath.Dir(grandchild), filepath.Base(grandchild), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]interface{}{"a": float64(1), "b": float64(20), "c": float64(30), "d": float64(400)}
	if !reflect.DeepEqual(result.Obj, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func writeTempJSON(t *testing.T, dir, name string, data string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(data), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

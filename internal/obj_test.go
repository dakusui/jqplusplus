package internal

import (
	"reflect"
	"testing"
)

func TestPutAtPath(t *testing.T) {
	obj := map[string]any{"a": "Hello", "b": "X"}

	PutAtPath(obj, []any{"xyz"}, "XYZ")

	expected := map[string]any{"a": "Hello", "b": "X", "xyz": "XYZ"}
	if !reflect.DeepEqual(expected, obj) {
		t.Errorf("Expected '%s', but got '%s'", expected, obj)
	}
}

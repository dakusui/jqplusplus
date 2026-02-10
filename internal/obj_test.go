package internal

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
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

// Generated test for Paths function
func TestPaths(t *testing.T) {
	obj := map[string]any{
		"arr": []any{
			map[string]any{
				"k": map[string]any{
					"v": "Hello!",
				},
				"content": map[string]any{
					"k1": "v1",
					"k2": "v2",
				},
			}}}

	result := Map[[]any, string](
		Paths(obj, func([]any) bool { return true }),
		func(anies []any) string {
			ret, err := PathArrayToPathExpression(anies)
			if err != nil {
				panic(fmt.Sprintf("Test fails: %v", err))
			}
			return ret
		})

	// Assuming the order is the same
	mustBeContained := ".arr[0].k.v"
	if !slices.Contains(result, mustBeContained) {
		t.Errorf("Expected '%s', but got '%s'", mustBeContained, result)
	}
}

func TestStringEntries(t *testing.T) {
	obj := map[string]any{
		"arr": []any{
			map[string]any{
				"k": map[string]any{
					"v": "Hello!",
				},
				"content": map[string]any{
					"k1": "v1",
					"k2": "v2",
				},
			}}}

	result := StringEntries(obj, func(string) bool { return true })

	// Assuming the order is the same

	for _, e := range result {
		fmt.Println(e)
	}
	/*
		mustBeContained := ".arr[0].k.v"
		if !slices.Contains(result, mustBeContained) {
			t.Errorf("Expected '%s', but got '%s'", mustBeContained, result)
		}

	*/
}

func TestStringEntries_StartingWithPrefixes(t *testing.T) {
	obj := map[string]any{
		"arr": []any{
			map[string]any{
				"k": map[string]any{
					"v": "eval:string:ref(parentof($cur) + [\"content\"])",
				},
				"content": map[string]any{
					"k1": "v1",
					"k2": "v2",
				},
			}}}

	result := StringEntries(obj, func(v string) bool {
		return strings.HasPrefix(v, "raw:") || strings.HasPrefix(v, "eval:")
	})

	// Assuming the order is the same

	for _, e := range result {
		fmt.Println(e)
	}
	/*
		mustBeContained := ".arr[0].k.v"
		if !slices.Contains(result, mustBeContained) {
			t.Errorf("Expected '%s', but got '%s'", mustBeContained, result)
		}

	*/
}

func TestDeepCopy_NoCycle(t *testing.T) {
	orig := map[string]any{
		"a": 1,
		"b": []any{
			map[string]any{"k": "v"},
		},
	}

	copied := DeepCopy(orig).(map[string]any)

	if !reflect.DeepEqual(orig, copied) {
		t.Fatalf("copied value mismatch: expected %v, got %v", orig, copied)
	}

	// Mutate original and ensure copy does not change.
	orig["a"] = 2
	orig["b"].([]any)[0].(map[string]any)["k"] = "changed"

	if copied["a"].(int) != 1 {
		t.Fatalf("copy was affected by mutation of original: copied[a]=%v", copied["a"])
	}
	if copied["b"].([]any)[0].(map[string]any)["k"] != "v" {
		t.Fatalf("copy was affected by mutation of original nested value: copied[b][0][k]=%v", copied["b"].([]any)[0].(map[string]any)["k"])
	}
}

func TestDeepCopy_PanicsOnCyclicMap(t *testing.T) {
	m := map[string]any{}
	m["self"] = m

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected DeepCopy to panic on cyclic map, but it did not")
		}
	}()

	_ = DeepCopy(m)
}

func TestDeepCopy_PanicsOnCyclicSlice(t *testing.T) {
	var s []any
	s = append(s, &s)

	// Create a cycle: element 0 points to the slice itself through indirection.
	s[0] = s

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected DeepCopy to panic on cyclic slice, but it did not")
		}
	}()

	_ = DeepCopy(s)
}

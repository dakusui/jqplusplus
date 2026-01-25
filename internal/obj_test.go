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

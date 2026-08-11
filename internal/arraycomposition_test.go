package internal

import (
	"reflect"
	"strings"
	"testing"
)

func TestSuperSplice(t *testing.T) {
	parent := map[string]any{"key": []any{"p0", "p1"}}

	for name, child := range map[string][]any{
		"append":   []any{"$super", "tail"},
		"prepend":  []any{"head", "$super"},
		"wrap":     []any{"head", "$super", "tail"},
		"multiple": []any{"$super", "middle", "$super"},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := mergeObjects(parent, map[string]any{"key": child})
			if err != nil {
				t.Fatalf("merge failed: %v", err)
			}
			var expected []any
			switch name {
			case "append":
				expected = []any{"p0", "p1", "tail"}
			case "prepend":
				expected = []any{"head", "p0", "p1"}
			case "wrap":
				expected = []any{"head", "p0", "p1", "tail"}
			case "multiple":
				expected = []any{"p0", "p1", "middle", "p0", "p1"}
			}
			if !reflect.DeepEqual(got["key"], expected) {
				t.Fatalf("expected %v, got %v", expected, got["key"])
			}
		})
	}
}

func TestSuperSpliceCarriesDeltaAndDoesNotMutateParent(t *testing.T) {
	parentArray := []any{"p0"}
	carried, err := mergeObjects(map[string]any{}, map[string]any{"key": []any{"$super", "child"}})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if want := []any{"$super", "child"}; !reflect.DeepEqual(carried["key"], want) {
		t.Fatalf("expected carried delta %v, got %v", want, carried["key"])
	}
	if _, err := mergeObjects(map[string]any{"key": parentArray}, map[string]any{"key": []any{"$super", "child"}}); err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if want := []any{"p0"}; !reflect.DeepEqual(parentArray, want) {
		t.Fatalf("inherited array was mutated: %v", parentArray)
	}
}

func TestValidateArrayComposition(t *testing.T) {
	for name, tc := range map[string]struct {
		object map[string]any
		want   string
	}{
		"unresolved marker": {map[string]any{"key": []any{"$super"}}, "unresolved marker"},
		"object key":        {map[string]any{"$super": "value"}, "out of context"},
		"scalar value":      {map[string]any{"key": "$super"}, "out of context"},
		"nested array":      {map[string]any{"key": []any{[]any{"$super"}}}, "out of context"},
		"unknown marker":    {map[string]any{"key": []any{"$super?"}}, "unknown marker"},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateArrayComposition(tc.object)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
	if err := ValidateArrayComposition(map[string]any{"key": []any{"$supervisor", "raw:$super"}}); err != nil {
		t.Fatalf("ordinary and escaped strings must pass validation: %v", err)
	}
}

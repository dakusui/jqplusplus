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

func TestSuperPairing(t *testing.T) {
	parent := map[string]any{"servers": []any{
		map[string]any{"host": "a", "port": 80.0},
		map[string]any{"host": "b", "port": 80.0},
	}}
	child := map[string]any{"servers": []any{
		map[string]any{"host": "new"},
		"$super*",
		map[string]any{"tls": true},
		map[string]any{},
		map[string]any{"host": "tail"},
	}}
	got, err := mergeObjects(parent, child)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	want := []any{
		map[string]any{"host": "new"},
		map[string]any{"host": "a", "port": 80.0, "tls": true},
		map[string]any{"host": "b", "port": 80.0},
		map[string]any{"host": "tail"},
	}
	if !reflect.DeepEqual(got["servers"], want) {
		t.Fatalf("expected %v, got %v", want, got["servers"])
	}
}

func TestSuperPairingPairRules(t *testing.T) {
	t.Run("atoms are overridden", func(t *testing.T) {
		got, err := mergeObjects(
			map[string]any{"ports": []any{80.0, 443.0}},
			map[string]any{"ports": []any{"$super*", 8080.0}},
		)
		if err != nil {
			t.Fatalf("merge failed: %v", err)
		}
		if want := []any{8080.0, 443.0}; !reflect.DeepEqual(got["ports"], want) {
			t.Fatalf("expected %v, got %v", want, got["ports"])
		}
	})

	t.Run("unmarked arrays replace", func(t *testing.T) {
		got, err := mergeObjects(
			map[string]any{"items": []any{[]any{"base"}}},
			map[string]any{"items": []any{"$super*", []any{"child"}}},
		)
		if err != nil {
			t.Fatalf("merge failed: %v", err)
		}
		if want := []any{[]any{"child"}}; !reflect.DeepEqual(got["items"], want) {
			t.Fatalf("expected %v, got %v", want, got["items"])
		}
	})

	t.Run("marked arrays recurse", func(t *testing.T) {
		got, err := mergeObjects(
			map[string]any{"items": []any{[]any{"base"}}},
			map[string]any{"items": []any{"$super*", []any{"$super", "child"}}},
		)
		if err != nil {
			t.Fatalf("merge failed: %v", err)
		}
		if want := []any{[]any{"base", "child"}}; !reflect.DeepEqual(got["items"], want) {
			t.Fatalf("expected %v, got %v", want, got["items"])
		}
	})

	for name, tc := range map[string]struct {
		parent any
		child  any
	}{
		"object and array": {map[string]any{}, []any{}},
		"array and atom":   {[]any{}, "atom"},
		"atom and object":  {"atom", map[string]any{}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := mergeObjects(
				map[string]any{"key": []any{tc.parent}},
				map[string]any{"key": []any{"$super*", tc.child}},
			)
			if err == nil || !strings.Contains(err.Error(), "cross-kind pair") {
				t.Fatalf("expected cross-kind-pair error, got %v", err)
			}
		})
	}
}

func TestSuperPairingRejectsIncompatibleDeltas(t *testing.T) {
	cases := []struct {
		name   string
		parent []any
		child  []any
		want   string
	}{
		{"mixed markers", []any{"base"}, []any{"$super", "$super*"}, "cannot be mixed"},
		{"duplicate pairing marker", []any{"base"}, []any{"$super*", "$super*"}, "at most once"},
		{"pairing over splice", []any{"$super", "base"}, []any{"$super*", "child"}, "cannot compose"},
		{"splice over pairing", []any{"$super*", "base"}, []any{"$super", "child"}, "cannot compose"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mergeObjects(map[string]any{"key": tc.parent}, map[string]any{"key": tc.child})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

package internal

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func jsonObj(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("bad test fixture %q: %v", s, err)
	}
	return m
}

// mustMergeNodes composes child over parent and fails the test on error.
func mustMergeNodes(t *testing.T, parent, child map[string]any) map[string]any {
	t.Helper()
	result, err := MergeNodes(parent, child)
	if err != nil {
		t.Fatalf("MergeNodes failed: %v", err)
	}
	return result
}

func TestMergeNodes_ArrayComposition(t *testing.T) {
	tests := []struct {
		name     string
		parent   string
		child    string
		expected string
	}{
		{
			name:     "unmarked child array replaces inherited array",
			parent:   `{"a": [1, 2, 3]}`,
			child:    `{"a": [4]}`,
			expected: `{"a": [4]}`,
		},
		{
			name:     "unmarked child array replaces inherited marked array",
			parent:   `{"a": ["$super", 1]}`,
			child:    `{"a": [4]}`,
			expected: `{"a": [4]}`,
		},
		{
			name:     "child atom replaces inherited marked array",
			parent:   `{"a": ["$super", 1]}`,
			child:    `{"a": "x"}`,
			expected: `{"a": "x"}`,
		},
		{
			name:     "splice append",
			parent:   `{"a": [1, 2]}`,
			child:    `{"a": ["$super", 3]}`,
			expected: `{"a": [1, 2, 3]}`,
		},
		{
			name:     "splice prepend",
			parent:   `{"a": [1, 2]}`,
			child:    `{"a": [0, "$super"]}`,
			expected: `{"a": [0, 1, 2]}`,
		},
		{
			name:     "splice wrap",
			parent:   `{"a": [1, 2]}`,
			child:    `{"a": [0, "$super", 3]}`,
			expected: `{"a": [0, 1, 2, 3]}`,
		},
		{
			name:     "splice at multiple positions",
			parent:   `{"a": [1]}`,
			child:    `{"a": ["$super", "x", "$super"]}`,
			expected: `{"a": [1, "x", 1]}`,
		},
		{
			name:     "splice with empty inherited array",
			parent:   `{"a": []}`,
			child:    `{"a": ["$super", 1]}`,
			expected: `{"a": [1]}`,
		},
		{
			name:     "token inside nested inner array is inert during splice",
			parent:   `{"a": [1]}`,
			child:    `{"a": ["$super", [2, "raw:$super"]]}`,
			expected: `{"a": [1, [2, "raw:$super"]]}`,
		},
		{
			name:     "splice delta carries when inherited key is absent",
			parent:   `{"b": 1}`,
			child:    `{"a": ["$super", 1]}`,
			expected: `{"a": ["$super", 1], "b": 1}`,
		},
		{
			name:     "splice child composes over inherited splice delta",
			parent:   `{"a": ["p", "$super"]}`,
			child:    `{"a": ["$super", "c"]}`,
			expected: `{"a": ["p", "$super", "c"]}`,
		},
		{
			name:     "splice works on arrays nested inside merged objects",
			parent:   `{"o": {"a": [1]}}`,
			child:    `{"o": {"a": ["$super", 2]}}`,
			expected: `{"o": {"a": [1, 2]}}`,
		},
		{
			name:     "pairwise basic index-wise merge",
			parent:   `{"a": [{"x": 1}, {"y": 2}]}`,
			child:    `{"a": ["$super*", {"x": 10}, {"z": 3}]}`,
			expected: `{"a": [{"x": 10}, {"y": 2, "z": 3}]}`,
		},
		{
			name:     "pairwise literal prefix",
			parent:   `{"a": [{"x": 1}]}`,
			child:    `{"a": ["new", "$super*", {"x": 2}]}`,
			expected: `{"a": ["new", {"x": 2}]}`,
		},
		{
			name:     "pairwise inherited extras survive after merged region",
			parent:   `{"a": [{"x": 1}, {"y": 2}, {"z": 3}]}`,
			child:    `{"a": ["$super*", {"x": 9}]}`,
			expected: `{"a": [{"x": 9}, {"y": 2}, {"z": 3}]}`,
		},
		{
			name:     "pairwise child extras become a literal tail",
			parent:   `{"a": [{"x": 1}]}`,
			child:    `{"a": ["$super*", {"x": 9}, "tail"]}`,
			expected: `{"a": [{"x": 9}, "tail"]}`,
		},
		{
			name:     "pairwise empty-object pad keeps the inherited element",
			parent:   `{"a": [{"x": 1}, {"y": 2}]}`,
			child:    `{"a": ["$super*", {}, {"y": 9}]}`,
			expected: `{"a": [{"x": 1}, {"y": 9}]}`,
		},
		{
			name:     "pairwise atom pair lets the child win",
			parent:   `{"a": [1, 2]}`,
			child:    `{"a": ["$super*", 10]}`,
			expected: `{"a": [10, 2]}`,
		},
		{
			name:     "pairwise null pair is an atom pair",
			parent:   `{"a": [null]}`,
			child:    `{"a": ["$super*", 5]}`,
			expected: `{"a": [5]}`,
		},
		{
			name:     "pairwise unmarked array pair replaces",
			parent:   `{"a": [[1, 2]]}`,
			child:    `{"a": ["$super*", [3]]}`,
			expected: `{"a": [[3]]}`,
		},
		{
			name:     "pairwise marked array pair recurses into splice",
			parent:   `{"a": [[1, 2]]}`,
			child:    `{"a": ["$super*", ["$super", 3]]}`,
			expected: `{"a": [[1, 2, 3]]}`,
		},
		{
			name:     "pairwise delta carries when inherited key is absent",
			parent:   `{"b": 1}`,
			child:    `{"a": ["$super*", {"x": 1}]}`,
			expected: `{"a": ["$super*", {"x": 1}], "b": 1}`,
		},
		{
			name:     "pairwise expresses prepend plus override plus append at once",
			parent:   `{"a": [{"x": 1}, {"y": 2}]}`,
			child:    `{"a": ["head", "$super*", {"x": 9}, {}, "tail"]}`,
			expected: `{"a": ["head", {"x": 9}, {"y": 2}, "tail"]}`,
		},
		{
			name:     "reserved-namespace lookalike strings are literals during merge",
			parent:   `{"a": [1]}`,
			child:    `{"a": ["$super", "$supervisor"]}`,
			expected: `{"a": [1, "$supervisor"]}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := mustMergeNodes(t, jsonObj(t, tc.parent), jsonObj(t, tc.child))
			expected := jsonObj(t, tc.expected)
			if !reflect.DeepEqual(result, expected) {
				t.Errorf("expected %v, got %v", expected, result)
			}
		})
	}
}

func TestMergeNodes_ArrayCompositionErrors(t *testing.T) {
	tests := []struct {
		name        string
		parent      string
		child       string
		expectedMsg string
	}{
		{
			name:        "splice child over non-array inherited value",
			parent:      `{"a": {"x": 1}}`,
			child:       `{"a": ["$super", 1]}`,
			expectedMsg: "requires the inherited value to be an array, but got object",
		},
		{
			name:        "splice child over inherited null",
			parent:      `{"a": null}`,
			child:       `{"a": ["$super", 1]}`,
			expectedMsg: "requires the inherited value to be an array, but got atom",
		},
		{
			name:        "pairwise child over non-array inherited value",
			parent:      `{"a": "text"}`,
			child:       `{"a": ["$super*", 1]}`,
			expectedMsg: "requires the inherited value to be an array, but got atom",
		},
		{
			name:        "splice child over inherited pairwise delta",
			parent:      `{"a": ["$super*", {"x": 1}]}`,
			child:       `{"a": ["$super", {"y": 2}]}`,
			expectedMsg: "cannot splice into an inherited pairwise delta",
		},
		{
			name:        "pairwise child over inherited splice delta",
			parent:      `{"a": ["$super", 1]}`,
			child:       `{"a": ["$super*", 2]}`,
			expectedMsg: "cannot pairwise-merge with an inherited marked array",
		},
		{
			name:        "pairwise child over inherited pairwise delta",
			parent:      `{"a": ["$super*", 1]}`,
			child:       `{"a": ["$super*", 2]}`,
			expectedMsg: "cannot pairwise-merge with an inherited marked array",
		},
		{
			name:        "both tokens in the same child array",
			parent:      `{"a": [1]}`,
			child:       `{"a": ["$super", "$super*"]}`,
			expectedMsg: "cannot appear in the same array",
		},
		{
			name:        "two pairwise tokens in one array",
			parent:      `{"a": [1]}`,
			child:       `{"a": ["$super*", 1, "$super*"]}`,
			expectedMsg: "at most one",
		},
		{
			name:        "invalid marking on the inherited side",
			parent:      `{"a": ["$super", "$super*"]}`,
			child:       `{"a": ["$super", 1]}`,
			expectedMsg: "cannot appear in the same array",
		},
		{
			name:        "pairwise cross-kind pair object with atom",
			parent:      `{"a": [{"x": 1}]}`,
			child:       `{"a": ["$super*", 5]}`,
			expectedMsg: "cannot merge object with atom",
		},
		{
			name:        "pairwise cross-kind pair array with object",
			parent:      `{"a": [[1]]}`,
			child:       `{"a": ["$super*", {"x": 1}]}`,
			expectedMsg: "cannot merge array with object",
		},
		{
			name:        "pairwise empty-object pad over atom is cross-kind",
			parent:      `{"a": [1]}`,
			child:       `{"a": ["$super*", {}]}`,
			expectedMsg: "cannot merge atom with object",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := MergeNodes(jsonObj(t, tc.parent), jsonObj(t, tc.child))
			if err == nil {
				t.Fatalf("expected an error containing %q, got none", tc.expectedMsg)
			}
			if !strings.Contains(err.Error(), tc.expectedMsg) {
				t.Errorf("expected error containing %q, got %q", tc.expectedMsg, err.Error())
			}
		})
	}
}

// Splice-delta composition must be associative: composing child over parent
// and then grounding over base must equal grounding parent over base first
// and composing child on top.
func TestMergeNodes_SpliceCompositionIsAssociative(t *testing.T) {
	base := jsonObj(t, `{"a": ["b0"]}`)
	parent := jsonObj(t, `{"a": ["$super", "p"]}`)
	child := jsonObj(t, `{"a": ["$super", "c"]}`)

	left := mustMergeNodes(t, mustMergeNodes(t, base, parent), child)
	right := mustMergeNodes(t, base, mustMergeNodes(t, parent, child))
	expected := jsonObj(t, `{"a": ["b0", "p", "c"]}`)

	if !reflect.DeepEqual(left, expected) {
		t.Errorf("(child over (parent over base)): expected %v, got %v", expected, left)
	}
	if !reflect.DeepEqual(right, expected) {
		t.Errorf("((child over parent) over base): expected %v, got %v", expected, right)
	}
}

// Composition must never mutate its inputs: inherited arrays live in the node
// pool cache and are shared between merge sites.
func TestMergeNodes_DoesNotMutateInputs(t *testing.T) {
	parent := jsonObj(t, `{"a": [1, 2], "b": [{"x": 1}]}`)
	child := jsonObj(t, `{"a": ["$super", 3], "b": ["$super*", {"y": 2}]}`)
	parentCopy := DeepCopyAs(parent)
	childCopy := DeepCopyAs(child)

	result := mustMergeNodes(t, parent, child)

	if !reflect.DeepEqual(parent, parentCopy) {
		t.Errorf("parent was mutated: %v", parent)
	}
	if !reflect.DeepEqual(child, childCopy) {
		t.Errorf("child was mutated: %v", child)
	}
	result["a"].([]any)[0] = "poisoned"
	result["b"].([]any)[0].(map[string]any)["x"] = "poisoned"
	if !reflect.DeepEqual(parent, parentCopy) {
		t.Errorf("mutating the result reached the cached parent: %v", parent)
	}
}

func TestValidateArrayCompositions(t *testing.T) {
	valid := []struct {
		name string
		obj  string
	}{
		{"no arrays", `{"a": {"b": 1}}`},
		{"plain array", `{"a": [1, "x", null]}`},
		{"raw-escaped token", `{"a": ["raw:$super", "raw:$super*"]}`},
		{"identifier extension is a literal", `{"a": ["$supervisor", "$superuser_home", "$super_"]}`},
		{"token-like object key is out of scope", `{"$super": [1], "a": [{"$super*": 1}]}`},
		{"token in a string value outside an array is a literal", `{"a": "$super"}`},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			if err := ValidateArrayCompositions(jsonObj(t, tc.obj)); err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}

	invalid := []struct {
		name        string
		obj         string
		expectedMsg string
	}{
		{"dangling splice token", `{"a": ["$super", 1]}`, `unresolved token "$super"`},
		{"dangling pairwise token", `{"a": [1, "$super*"]}`, `unresolved token "$super*"`},
		{"dangling token in nested array", `{"a": [[1, "$super"]]}`, `unresolved token "$super"`},
		{"dangling token deep in an object", `{"a": {"b": {"c": ["$super"]}}}`, "at '.a.b.c[0]'"},
		{"reserved lenient splice spelling", `{"a": ["$super?"]}`, `unknown token "$super?"`},
		{"reserved lenient pairwise spelling", `{"a": ["$super*?"]}`, `unknown token "$super*?"`},
		{"reserved future keyed syntax", `{"a": ["$super*(name)"]}`, `unknown token "$super*(name)"`},
		{"reserved spelling with a space", `{"a": ["$super x"]}`, "unknown token"},
		{"doubled star", `{"a": ["$super**"]}`, `unknown token "$super**"`},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			err := ValidateArrayCompositions(jsonObj(t, tc.obj))
			if err == nil {
				t.Fatalf("expected an error containing %q, got none", tc.expectedMsg)
			}
			if !strings.Contains(err.Error(), tc.expectedMsg) {
				t.Errorf("expected error containing %q, got %q", tc.expectedMsg, err.Error())
			}
		})
	}
}

// The sweep walks object keys in sorted order, so the first reported error is
// deterministic.
func TestValidateArrayCompositions_DeterministicFirstError(t *testing.T) {
	obj := jsonObj(t, `{"zz": ["$super"], "aa": ["$super*"], "mm": ["$super?"]}`)
	for range 16 {
		err := ValidateArrayCompositions(obj)
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "at '.aa[0]'") {
			t.Fatalf("expected the error at '.aa[0]' to be reported first, got %q", err.Error())
		}
	}
}

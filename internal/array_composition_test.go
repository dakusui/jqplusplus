package internal

import (
	"reflect"
	"strings"
	"testing"
)

func TestMergeObjects_UnmarkedArrayReplacesInheritedArray(t *testing.T) {
	parent := map[string]any{"values": []any{"parent"}}
	child := map[string]any{"values": []any{"child"}}

	got, err := mergeObjects(parent, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"values": []any{"child"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeObjects_SuperSplicesAtEveryOccurrence(t *testing.T) {
	parent := map[string]any{"values": []any{"p1", "p2"}}
	tests := []struct {
		name  string
		child []any
		want  []any
	}{
		{name: "append", child: []any{superSpliceToken, "after"}, want: []any{"p1", "p2", "after"}},
		{name: "prepend", child: []any{"before", superSpliceToken}, want: []any{"before", "p1", "p2"}},
		{name: "wrap", child: []any{"before", superSpliceToken, "after"}, want: []any{"before", "p1", "p2", "after"}},
		{name: "multiple", child: []any{superSpliceToken, "middle", superSpliceToken}, want: []any{"p1", "p2", "middle", "p1", "p2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeObjects(parent, map[string]any{"values": tt.child})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got["values"], tt.want) {
				t.Fatalf("got %v, want %v", got["values"], tt.want)
			}
		})
	}
}

func TestMergeObjects_SuperDoesNotDepthSpliceNestedArrays(t *testing.T) {
	got, err := mergeObjects(
		map[string]any{"values": []any{"parent"}},
		map[string]any{"values": []any{superSpliceToken, []any{superSpliceToken}}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{"parent", []any{superSpliceToken}}
	if !reflect.DeepEqual(got["values"], want) {
		t.Fatalf("got %v, want %v", got["values"], want)
	}
}

func TestMergeObjects_ArrayCompositionBuildsFreshSlices(t *testing.T) {
	parentSplice := []any{"parent"}
	childSplice := []any{superSpliceToken, "child"}
	spliced, err := mergeValues(parentSplice, childSplice, MergePolicyDefault)
	if err != nil {
		t.Fatalf("splice: unexpected error: %v", err)
	}
	spliced.([]any)[0] = "changed-parent"
	spliced.([]any)[1] = "changed-child"
	if !reflect.DeepEqual(parentSplice, []any{"parent"}) ||
		!reflect.DeepEqual(childSplice, []any{superSpliceToken, "child"}) {
		t.Fatalf("splice aliases an input: parent=%v child=%v", parentSplice, childSplice)
	}

	parentMerge := []any{"parent", "parent-tail"}
	childMerge := []any{superMergeToken, "child"}
	paired, err := mergeValues(parentMerge, childMerge, MergePolicyDefault)
	if err != nil {
		t.Fatalf("pairwise merge: unexpected error: %v", err)
	}
	paired.([]any)[0] = "changed-child"
	paired.([]any)[1] = "changed-parent"
	if !reflect.DeepEqual(parentMerge, []any{"parent", "parent-tail"}) ||
		!reflect.DeepEqual(childMerge, []any{superMergeToken, "child"}) {
		t.Fatalf("pairwise merge aliases an input: parent=%v child=%v", parentMerge, childMerge)
	}
}

func TestMergeObjects_SuperDeltasComposeAssociatively(t *testing.T) {
	base := map[string]any{"values": []any{"base"}}
	firstDelta := map[string]any{"values": []any{"first", superSpliceToken}}
	secondDelta := map[string]any{"values": []any{superSpliceToken, "second"}}

	baseThenFirst, err := mergeObjects(base, firstDelta)
	if err != nil {
		t.Fatalf("compose base and first delta: %v", err)
	}
	left, err := mergeObjects(baseThenFirst, secondDelta)
	if err != nil {
		t.Fatalf("compose left side: %v", err)
	}
	composedDelta, err := mergeObjects(firstDelta, secondDelta)
	if err != nil {
		t.Fatalf("compose deltas: %v", err)
	}
	right, err := mergeObjects(base, composedDelta)
	if err != nil {
		t.Fatalf("ground composed delta: %v", err)
	}

	want := map[string]any{"values": []any{"first", "base", "second"}}
	if !reflect.DeepEqual(left, right) || !reflect.DeepEqual(left, want) {
		t.Fatalf("left=%v, right=%v, want=%v", left, right, want)
	}
}

func TestMergeObjects_SuperMergePairsValuesByIndex(t *testing.T) {
	parentArray := []any{
		map[string]any{"inherited": true},
		[]any{"inherited-array"},
		"inherited-atom",
		map[string]any{"kept": true},
		"inherited-tail",
	}
	childArray := []any{
		"literal-prefix",
		superMergeToken,
		map[string]any{"child": true},
		[]any{"child-array"},
		"child-atom",
		map[string]any{},
	}

	got, err := mergeObjects(
		map[string]any{"values": parentArray},
		map[string]any{"values": childArray},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{
		"literal-prefix",
		map[string]any{"inherited": true, "child": true},
		[]any{"child-array"},
		"child-atom",
		map[string]any{"kept": true},
		"inherited-tail",
	}
	if !reflect.DeepEqual(got["values"], want) {
		t.Fatalf("got %v, want %v", got["values"], want)
	}
	if !reflect.DeepEqual(parentArray, []any{
		map[string]any{"inherited": true},
		[]any{"inherited-array"},
		"inherited-atom",
		map[string]any{"kept": true},
		"inherited-tail",
	}) {
		t.Fatalf("parent array was mutated: %v", parentArray)
	}
}

func TestMergeObjects_SuperMergeKeepsChildTail(t *testing.T) {
	got, err := mergeObjects(
		map[string]any{"values": []any{"parent"}},
		map[string]any{"values": []any{superMergeToken, "override", "child-tail"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{"override", "child-tail"}
	if !reflect.DeepEqual(got["values"], want) {
		t.Fatalf("got %v, want %v", got["values"], want)
	}
}

func TestMergeObjects_SuperMergeRecursesIntoMarkedNestedArray(t *testing.T) {
	got, err := mergeObjects(
		map[string]any{"values": []any{[]any{map[string]any{"a": 1}}}},
		map[string]any{"values": []any{superMergeToken, []any{superMergeToken, map[string]any{"b": 2}}}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{[]any{map[string]any{"a": 1, "b": 2}}}
	if !reflect.DeepEqual(got["values"], want) {
		t.Fatalf("got %v, want %v", got["values"], want)
	}
}

func TestMergeObjects_ArrayCompositionErrors(t *testing.T) {
	tests := []struct {
		name   string
		parent []any
		child  any
		want   string
	}{
		{name: "splice requires array", parent: nil, child: []any{superSpliceToken}, want: "expected array"},
		{name: "cross-kind object to array", parent: []any{map[string]any{}}, child: []any{superMergeToken, []any{}}, want: "cross-kind merge"},
		{name: "cross-kind array to object", parent: []any{[]any{}}, child: []any{superMergeToken, map[string]any{}}, want: "cross-kind merge"},
		{name: "mixed tokens", parent: []any{"p"}, child: []any{superSpliceToken, superMergeToken}, want: "cannot be used in the same array"},
		{name: "double merge token", parent: []any{"p"}, child: []any{superMergeToken, superMergeToken}, want: "may appear only once"},
		{name: "pairwise delta under splice delta", parent: []any{superSpliceToken}, child: []any{superMergeToken}, want: "cannot compose marked arrays"},
		{name: "splice delta under pairwise delta", parent: []any{superMergeToken}, child: []any{superSpliceToken}, want: "cannot compose marked arrays"},
		{name: "pairwise deltas", parent: []any{superMergeToken}, child: []any{superMergeToken}, want: "cannot compose marked arrays"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inherited any = tt.parent
			if tt.name == "splice requires array" {
				inherited = "not-an-array"
			}
			_, err := mergeValues(inherited, tt.child, MergePolicyDefault)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got error %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestGroundArrayCompositions(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr string
	}{
		{name: "dangling splice", value: superSpliceToken, wantErr: "unresolved array-composition token"},
		{name: "dangling merge", value: superMergeToken, wantErr: "unresolved array-composition token"},
		{name: "reserved optional splice", value: "$super?", wantErr: "unknown array-composition token"},
		{name: "reserved future selector", value: "$super*(name)", wantErr: "unknown array-composition token"},
		{name: "reserved whitespace", value: "$super later", wantErr: "unknown array-composition token"},
		{name: "identifier suffix", value: "$supervisor"},
		{name: "identifier underscore", value: "$super_user"},
		{name: "raw escape", value: "raw:$super"},
		{name: "eval result immune", value: "eval:string:\"$super\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GroundArrayCompositions(map[string]any{"outer": []any{[]any{tt.value}}})
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("got error %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

package internal

import (
	"reflect"
	"strings"
	"testing"
)

func mustMerge(t *testing.T, parent, child map[string]any) map[string]any {
	t.Helper()
	result, err := mergeObjects(parent, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return result
}

func mustFailMerge(t *testing.T, parent, child map[string]any, wantSubstr string) {
	t.Helper()
	_, err := mergeObjects(parent, child)
	if err == nil {
		t.Fatalf("expected an error containing %q, got none", wantSubstr)
	}
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("expected error containing %q, got: %v", wantSubstr, err)
	}
}

func TestSuperSplice(t *testing.T) {
	parent := map[string]any{"key": []any{"p0", "p1"}}

	t.Run("append", func(t *testing.T) {
		result := mustMerge(t, parent, map[string]any{"key": []any{"$super", "c0"}})
		expected := []any{"p0", "p1", "c0"}
		if !reflect.DeepEqual(result["key"], expected) {
			t.Errorf("expected %v, got %v", expected, result["key"])
		}
	})

	t.Run("prepend", func(t *testing.T) {
		result := mustMerge(t, parent, map[string]any{"key": []any{"c0", "$super"}})
		expected := []any{"c0", "p0", "p1"}
		if !reflect.DeepEqual(result["key"], expected) {
			t.Errorf("expected %v, got %v", expected, result["key"])
		}
	})

	t.Run("wrap and multiple occurrences", func(t *testing.T) {
		result := mustMerge(t, parent, map[string]any{"key": []any{"a", "$super", "b", "$super"}})
		expected := []any{"a", "p0", "p1", "b", "p0", "p1"}
		if !reflect.DeepEqual(result["key"], expected) {
			t.Errorf("expected %v, got %v", expected, result["key"])
		}
	})

	t.Run("unmarked child array still replaces", func(t *testing.T) {
		result := mustMerge(t, parent, map[string]any{"key": []any{"only"}})
		expected := []any{"only"}
		if !reflect.DeepEqual(result["key"], expected) {
			t.Errorf("expected %v, got %v", expected, result["key"])
		}
	})

	t.Run("absent parent key carries the delta", func(t *testing.T) {
		result := mustMerge(t, map[string]any{}, map[string]any{"key": []any{"$super", "c0"}})
		expected := []any{"$super", "c0"}
		if !reflect.DeepEqual(result["key"], expected) {
			t.Errorf("expected %v, got %v", expected, result["key"])
		}
	})

	t.Run("delta composition is associative", func(t *testing.T) {
		base := map[string]any{"key": []any{"z"}}
		deltaA := map[string]any{"key": []any{"$super", "a"}}
		deltaC := map[string]any{"key": []any{"$super", "c"}}
		expected := []any{"z", "a", "c"}

		// C over (A over base)
		left := mustMerge(t, mustMerge(t, base, deltaA), deltaC)
		// (C over A) over base — the inner merge carries an unresolved delta
		right := mustMerge(t, base, mustMerge(t, deltaA, deltaC))
		if !reflect.DeepEqual(left["key"], expected) || !reflect.DeepEqual(right["key"], expected) {
			t.Errorf("expected %v both ways, got %v and %v", expected, left["key"], right["key"])
		}
	})

	t.Run("non-array parent is a kind error", func(t *testing.T) {
		mustFailMerge(t, map[string]any{"key": "scalar"}, map[string]any{"key": []any{"$super"}},
			"a merge is only defined between values of the same kind")
	})

	t.Run("parent slice is not mutated", func(t *testing.T) {
		shared := []any{"p0", "p1"}
		parent := map[string]any{"key": shared}
		_ = mustMerge(t, parent, map[string]any{"key": []any{"$super", "c0"}})
		if !reflect.DeepEqual(shared, []any{"p0", "p1"}) {
			t.Errorf("parent slice was mutated: %v", shared)
		}
	})
}

func TestSuperPairwiseMerge(t *testing.T) {
	parent := map[string]any{"servers": []any{
		map[string]any{"host": "a.example.com", "port": 80.0, "tls": false},
		map[string]any{"host": "b.example.com", "port": 80.0},
	}}

	t.Run("pairwise object merge with prefix and parent extras", func(t *testing.T) {
		child := map[string]any{"servers": []any{
			map[string]any{"host": "new.example.com"},
			"$super*",
			map[string]any{"tls": true},
		}}
		result := mustMerge(t, parent, child)
		expected := []any{
			map[string]any{"host": "new.example.com"},
			map[string]any{"host": "a.example.com", "port": 80.0, "tls": true},
			map[string]any{"host": "b.example.com", "port": 80.0},
		}
		if !reflect.DeepEqual(result["servers"], expected) {
			t.Errorf("expected %v, got %v", expected, result["servers"])
		}
	})

	t.Run("empty object is a keep-as-is pad", func(t *testing.T) {
		child := map[string]any{"servers": []any{"$super*", map[string]any{}, map[string]any{"port": 8080.0}}}
		result := mustMerge(t, parent, child)
		expected := []any{
			map[string]any{"host": "a.example.com", "port": 80.0, "tls": false},
			map[string]any{"host": "b.example.com", "port": 8080.0},
		}
		if !reflect.DeepEqual(result["servers"], expected) {
			t.Errorf("expected %v, got %v", expected, result["servers"])
		}
	})

	t.Run("child queue extras become a literal tail", func(t *testing.T) {
		child := map[string]any{"servers": []any{"$super*", map[string]any{}, map[string]any{}, map[string]any{"host": "c.example.com"}}}
		result := mustMerge(t, parent, child)
		got := result["servers"].([]any)
		if len(got) != 3 || !reflect.DeepEqual(got[2], map[string]any{"host": "c.example.com"}) {
			t.Errorf("expected literal tail, got %v", got)
		}
	})

	t.Run("atom pairs take the child value", func(t *testing.T) {
		result := mustMerge(t,
			map[string]any{"ports": []any{80.0, 443.0}},
			map[string]any{"ports": []any{"$super*", 8080.0}})
		expected := []any{8080.0, 443.0}
		if !reflect.DeepEqual(result["ports"], expected) {
			t.Errorf("expected %v, got %v", expected, result["ports"])
		}
	})

	t.Run("cross-kind pair is an error", func(t *testing.T) {
		mustFailMerge(t,
			map[string]any{"ports": []any{80.0}},
			map[string]any{"ports": []any{"$super*", map[string]any{"port": 8080.0}}},
			"a merge is only defined between values of the same kind")
	})

	t.Run("mixing tokens is an error", func(t *testing.T) {
		mustFailMerge(t, parent, map[string]any{"servers": []any{"$super", "$super*"}},
			"cannot be mixed")
	})

	t.Run("double merge token is an error", func(t *testing.T) {
		mustFailMerge(t, parent, map[string]any{"servers": []any{"$super*", "$super*"}},
			"at most once")
	})

	t.Run("marked parent under pairwise merge is an error", func(t *testing.T) {
		mustFailMerge(t,
			map[string]any{"key": []any{"$super", "a"}},
			map[string]any{"key": []any{"$super*", "b"}},
			"unresolved composition tokens")
	})

	t.Run("nested marked array inside a paired object composes", func(t *testing.T) {
		parent := map[string]any{"items": []any{map[string]any{"tags": []any{"t0"}}}}
		child := map[string]any{"items": []any{"$super*", map[string]any{"tags": []any{"$super", "t1"}}}}
		result := mustMerge(t, parent, child)
		expected := []any{map[string]any{"tags": []any{"t0", "t1"}}}
		if !reflect.DeepEqual(result["items"], expected) {
			t.Errorf("expected %v, got %v", expected, result["items"])
		}
	})
}

func TestValidateArrayComposition(t *testing.T) {
	t.Run("dangling token is an error", func(t *testing.T) {
		err := ValidateArrayComposition(map[string]any{"key": []any{"$super", "x"}})
		if err == nil || !strings.Contains(err.Error(), "unresolved") {
			t.Errorf("expected unresolved-token error, got: %v", err)
		}
	})

	t.Run("dangling token nested deep is an error", func(t *testing.T) {
		err := ValidateArrayComposition(map[string]any{"a": map[string]any{"b": []any{[]any{"$super*"}}}})
		if err == nil || !strings.Contains(err.Error(), "unresolved") {
			t.Errorf("expected unresolved-token error, got: %v", err)
		}
	})

	t.Run("unknown token in reserved name space is an error", func(t *testing.T) {
		err := ValidateArrayComposition(map[string]any{"key": []any{"$super?"}})
		if err == nil || !strings.Contains(err.Error(), "unknown token") {
			t.Errorf("expected unknown-token error, got: %v", err)
		}
	})

	t.Run("identifier continuation is an ordinary literal", func(t *testing.T) {
		if err := ValidateArrayComposition(map[string]any{"key": []any{"$supervisor", "$superuser_home"}}); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("raw-escaped token passes validation", func(t *testing.T) {
		if err := ValidateArrayComposition(map[string]any{"key": []any{"raw:$super"}}); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("clean object passes", func(t *testing.T) {
		if err := ValidateArrayComposition(map[string]any{"key": []any{"a", 1.0, map[string]any{"b": []any{"c"}}}}); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

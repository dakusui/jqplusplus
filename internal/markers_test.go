package internal

import (
	"strings"
	"testing"
)

func TestClassifyMarker(t *testing.T) {
	t.Run("defined markers", func(t *testing.T) {
		for _, c := range []struct {
			in   string
			want MarkerKind
		}{
			{"$super", SpliceMarker},
			{"$super*", PairMarker},
		} {
			got, err := ClassifyMarker(c.in)
			if err != nil {
				t.Errorf("%q: unexpected error: %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("%q: expected %v, got %v", c.in, c.want, got)
			}
			if !got.IsMarker() {
				t.Errorf("%q: expected a marker", c.in)
			}
		}
	})

	t.Run("undefined spellings in the namespace are errors", func(t *testing.T) {
		for _, in := range []string{"$super[1:]", "$super?", "$super*?", "$super**", "$super("} {
			got, err := ClassifyMarker(in)
			if err == nil {
				t.Errorf("%q: expected an error, got %v", in, got)
				continue
			}
			if !strings.Contains(err.Error(), "unknown array composition marker") {
				t.Errorf("%q: unexpected message: %v", in, err)
			}
		}
	})

	t.Run("an identifier continuation is a different word", func(t *testing.T) {
		for _, in := range []string{"$supervisor", "$super_home", "$super1", "$superuser"} {
			got, err := ClassifyMarker(in)
			if err != nil {
				t.Errorf("%q: unexpected error: %v", in, err)
			}
			if got != NotAMarker {
				t.Errorf("%q: expected ordinary data, got %v", in, got)
			}
		}
	})

	t.Run("ordinary strings", func(t *testing.T) {
		for _, in := range []string{"", "super", "$sup", "a", "raw:$super", "eval:string:1"} {
			got, err := ClassifyMarker(in)
			if err != nil {
				t.Errorf("%q: unexpected error: %v", in, err)
			}
			if got != NotAMarker {
				t.Errorf("%q: expected ordinary data, got %v", in, got)
			}
		}
	})
}

func TestClassifyMarkerElement(t *testing.T) {
	t.Run("non-strings are never markers", func(t *testing.T) {
		for _, in := range []any{1, true, nil, []any{"$super"}, map[string]any{"$super": 1}} {
			got, err := ClassifyMarkerElement(in)
			if err != nil {
				t.Errorf("%v: unexpected error: %v", in, err)
			}
			if got != NotAMarker {
				t.Errorf("%v: expected ordinary data, got %v", in, got)
			}
		}
	})

	t.Run("strings are classified", func(t *testing.T) {
		got, err := ClassifyMarkerElement("$super")
		if err != nil || got != SpliceMarker {
			t.Errorf("expected the splice marker, got %v (%v)", got, err)
		}
	})
}

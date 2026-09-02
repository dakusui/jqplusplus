package internal

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Array composition markers. A marker is one of these exact strings written as
// a direct element of an array value, telling the inheritance stage how that
// array composes with the array it inherits.
const (
	// MarkerSplice substitutes the inherited array's elements at its position.
	MarkerSplice = "$super"
	// MarkerPair ends the literal prefix and begins index-wise pairing with the
	// inherited elements.
	MarkerPair = "$super*"
)

// markerNamespace is reserved: a string starting with it, followed by a
// character that cannot appear in an identifier, must be a defined marker or it
// is an error. That keeps undefined spellings such as "$super[1:]" available to
// gain meaning later without changing any working configuration.
const markerNamespace = "$super"

// MarkerKind classifies a string found in a direct array element position.
type MarkerKind int

const (
	// NotAMarker is ordinary data.
	NotAMarker MarkerKind = iota
	// SpliceMarker is MarkerSplice.
	SpliceMarker
	// PairMarker is MarkerPair.
	PairMarker
)

func (k MarkerKind) String() string {
	switch k {
	case SpliceMarker:
		return MarkerSplice
	case PairMarker:
		return MarkerPair
	default:
		return "not a marker"
	}
}

// IsMarker reports whether the kind denotes a marker rather than ordinary data.
func (k MarkerKind) IsMarker() bool {
	return k == SpliceMarker || k == PairMarker
}

// ClassifyMarker classifies a string in a direct array element position.
//
// A string equal to a defined marker is that marker. A string beginning with
// the reserved namespace followed by an identifier character is a different
// word, and so is ordinary data: "$supervisor" is not a malformed "$super".
// A string beginning with the namespace followed by anything else is an
// undefined spelling within it, and is an error.
func ClassifyMarker(s string) (MarkerKind, error) {
	switch s {
	case MarkerSplice:
		return SpliceMarker, nil
	case MarkerPair:
		return PairMarker, nil
	}
	if !strings.HasPrefix(s, markerNamespace) {
		return NotAMarker, nil
	}
	r, _ := utf8.DecodeRuneInString(s[len(markerNamespace):])
	if isIdentifierRune(r) {
		return NotAMarker, nil
	}
	return NotAMarker, fmt.Errorf("unknown array composition marker: %q; expected %q or %q", s, MarkerSplice, MarkerPair)
}

// ClassifyMarkerElement classifies an array element, which is a marker only if
// it is a string.
func ClassifyMarkerElement(v any) (MarkerKind, error) {
	s, ok := v.(string)
	if !ok {
		return NotAMarker, nil
	}
	return ClassifyMarker(s)
}

func isIdentifierRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// GroundMarkers reports any array composition marker that composition never
// resolved. It is the final pass over the top-level object, after node-level
// inheritance and before key-side evaluation.
//
// A marker still present at this point can never be bound to anything, so
// reporting it surfaces mistakes that a silent empty splice would hide: a
// mistyped key, a forgotten $extends, or a parent skipped by the optional-file
// marker. It also catches a marker written where nothing could ever bind it —
// inside an array nested in an unmarked array, or inside a value that pairing
// took whole.
//
// This runs on the top-level object only. Per-file cached resolutions keep
// their markers, because a fragment's delta is meant to survive until the
// document that includes it supplies an array to bind against.
//
// Strings produced by eval: expressions are not seen here, since value-side
// evaluation has not run yet. Marker classification is a check on what the
// author wrote, not on what the document renders.
func GroundMarkers(v any) error {
	return groundMarkers(v, nil)
}

func groundMarkers(v any, path []any) error {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err := groundMarkers(t[k], appendPathSegment(path, k)); err != nil {
				return err
			}
		}
	case []any:
		for i, e := range t {
			kind, err := ClassifyMarkerElement(e)
			if err != nil {
				return fmt.Errorf("%v: at '%s'", err, toPathExpression(path))
			}
			if kind.IsMarker() {
				return fmt.Errorf("unresolved array composition marker: %q: at '%s'", kind.String(), toPathExpression(path))
			}
			if err := groundMarkers(e, appendPathSegment(path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// appendPathSegment copies rather than appending in place, so that siblings do
// not share a backing array and overwrite each other's last segment.
func appendPathSegment(path []any, segment any) []any {
	p := make([]any, len(path), len(path)+1)
	copy(p, path)
	return append(p, segment)
}

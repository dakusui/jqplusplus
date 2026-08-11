package internal

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	superMarker     = "$super"
	superStarMarker = "$super*"
)

// GroundArrayComposition validates the reserved $super marker family after
// all file-level and node-level inheritance has been resolved. Cached nodes
// intentionally skip this pass so that a delta can remain pending until its
// including document provides an inherited array.
func GroundArrayComposition(obj map[string]any) error {
	return groundValue(obj, false)
}

func groundValue(value any, parentIsArray bool) error {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if err := validateMarkerOutsideArray(key); err != nil {
				return err
			}
			if err := groundValue(child, false); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range v {
			if stringValue, ok := child.(string); ok {
				if isRawString(stringValue) {
					continue
				}
				switch marker := classifyMarker(stringValue); marker {
				case superMarker, superStarMarker:
					if parentIsArray {
						return fmt.Errorf("array composition: marker %q is out of context", marker)
					}
					return fmt.Errorf("array composition: unresolved marker %q", marker)
				case "unknown":
					return fmt.Errorf("array composition: unknown marker %q", stringValue)
				}
				continue
			}
			if err := groundValue(child, true); err != nil {
				return err
			}
		}
	case string:
		if err := validateMarkerOutsideArray(v); err != nil {
			return err
		}
	}
	return nil
}

func validateMarkerOutsideArray(value string) error {
	if isRawString(value) {
		return nil
	}
	switch marker := classifyMarker(value); marker {
	case superMarker, superStarMarker:
		return fmt.Errorf("array composition: marker %q is out of context", marker)
	case "unknown":
		return fmt.Errorf("array composition: unknown marker %q", value)
	default:
		return nil
	}
}

func isRawString(value string) bool {
	return strings.HasPrefix(value, prefixRaw)
}

// classifyMarker returns the exact marker, "unknown" for a reserved but
// unsupported spelling, or "" for an ordinary string.
func classifyMarker(value string) string {
	switch value {
	case superMarker, superStarMarker:
		return value
	}
	if !strings.HasPrefix(value, superMarker) {
		return ""
	}
	suffix := strings.TrimPrefix(value, superMarker)
	if suffix == "" {
		return superMarker
	}
	first, _ := utf8.DecodeRuneInString(suffix)
	if isIdentifierRune(first) {
		return ""
	}
	return "unknown"
}

func isIdentifierRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

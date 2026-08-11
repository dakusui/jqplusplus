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

type arrayCompositionMode int

const (
	arrayCompositionNone arrayCompositionMode = iota
	arrayCompositionSplice
	arrayCompositionPair
)

type arrayComposition struct {
	mode  arrayCompositionMode
	index int
}

func (c arrayComposition) marker() string {
	switch c.mode {
	case arrayCompositionSplice:
		return superMarker
	case arrayCompositionPair:
		return superStarMarker
	default:
		return ""
	}
}

func parseArrayComposition(values []any) (arrayComposition, error) {
	superCount := 0
	superStarCount := 0
	superStarIndex := -1
	for index, value := range values {
		switch value {
		case superMarker:
			superCount++
		case superStarMarker:
			superStarCount++
			superStarIndex = index
		}
	}
	if superCount > 0 && superStarCount > 0 {
		return arrayComposition{}, fmt.Errorf("array composition: cannot mix \"$super\" and \"$super*\"")
	}
	if superStarCount > 1 {
		return arrayComposition{}, fmt.Errorf("array composition: \"$super*\" may appear at most once")
	}
	if superStarCount == 1 {
		return arrayComposition{mode: arrayCompositionPair, index: superStarIndex}, nil
	}
	if superCount > 0 {
		return arrayComposition{mode: arrayCompositionSplice}, nil
	}
	return arrayComposition{mode: arrayCompositionNone}, nil
}

func pairSuper(inherited, delta []any, markerIndex int) ([]any, error) {
	prefix := delta[:markerIndex]
	queue := delta[markerIndex+1:]
	length := len(inherited)
	if len(queue) > length {
		length = len(queue)
	}
	result := make([]any, 0, len(prefix)+length)
	result = append(result, prefix...)
	for index := 0; index < len(inherited) && index < len(queue); index++ {
		merged, err := mergePairedValues(inherited[index], queue[index], index)
		if err != nil {
			return nil, err
		}
		result = append(result, merged)
	}
	if len(inherited) > len(queue) {
		result = append(result, inherited[len(queue):]...)
	} else if len(queue) > len(inherited) {
		result = append(result, queue[len(inherited):]...)
	}
	return result, nil
}

func mergePairedValues(inherited, override any, index int) (any, error) {
	inheritedKind := compositionKindOf(inherited)
	overrideKind := compositionKindOf(override)
	if inheritedKind != overrideKind {
		return nil, fmt.Errorf("array composition: element %d pairs %s with %s", index, inheritedKind, overrideKind)
	}
	switch inheritedKind {
	case compositionKindObject:
		return mergeObjects(inherited.(map[string]any), override.(map[string]any))
	case compositionKindArray:
		return mergeInheritedValue(inherited, override)
	default:
		return override, nil
	}
}

type compositionKind string

const (
	compositionKindObject compositionKind = "object"
	compositionKindArray  compositionKind = "array"
	compositionKindAtom   compositionKind = "atom"
)

func compositionKindOf(value any) compositionKind {
	switch value.(type) {
	case map[string]any:
		return compositionKindObject
	case []any:
		return compositionKindArray
	default:
		return compositionKindAtom
	}
}

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
		if !parentIsArray {
			if _, err := parseArrayComposition(v); err != nil {
				return err
			}
		}
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

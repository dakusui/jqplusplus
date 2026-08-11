package internal

import (
	"fmt"
	"strings"
	"unicode"
)

const superToken = "$super"

// containsSuperToken reports whether arr contains a splice marker as a direct
// element. Markers in an inner array have no composition meaning.
func containsSuperToken(arr []any) bool {
	for _, element := range arr {
		if element == superToken {
			return true
		}
	}
	return false
}

// spliceSuperArray substitutes the parent elements at every splice marker.
// It always allocates a new slice so that a cached inherited array is never
// modified through the result.
func spliceSuperArray(parent, child []any) ([]any, error) {
	result := make([]any, 0, len(parent)+len(child))
	for _, element := range child {
		if element == superToken {
			result = append(result, parent...)
			continue
		}
		result = append(result, element)
	}
	return result, nil
}

type markerContext int

const (
	markerArrayElement markerContext = iota
	markerObjectKey
	markerScalarValue
	markerNestedArrayElement
	markerFilename
)

// ValidateArrayComposition grounds the fully resolved top-level object. A
// pending splice can no longer acquire an inherited array at this point, so it
// is an error. The same pass keeps the "$super" namespace extensible and
// rejects marker strings outside their one valid position.
func ValidateArrayComposition(obj map[string]any) error {
	return validateCompositionObject(obj, "$")
}

func validateCompositionObject(obj map[string]any, path string) error {
	for key, value := range obj {
		if err := validateSuperString(key, markerObjectKey); err != nil {
			return fmt.Errorf("%w at %s", err, path)
		}
		childPath := path + "." + key
		switch x := value.(type) {
		case map[string]any:
			if err := validateCompositionObject(x, childPath); err != nil {
				return err
			}
		case []any:
			if err := validateCompositionArray(x, childPath, true); err != nil {
				return err
			}
		case string:
			if err := validateSuperString(x, markerScalarValue); err != nil {
				return fmt.Errorf("%w at %s", err, childPath)
			}
		}
	}
	return nil
}

func validateCompositionArray(arr []any, path string, directObjectArray bool) error {
	for index, value := range arr {
		childPath := fmt.Sprintf("%s[%d]", path, index)
		switch x := value.(type) {
		case string:
			context := markerNestedArrayElement
			if directObjectArray {
				context = markerArrayElement
			}
			if err := validateSuperString(x, context); err != nil {
				return fmt.Errorf("%w at %s", err, childPath)
			}
		case map[string]any:
			if err := validateCompositionObject(x, childPath); err != nil {
				return err
			}
		case []any:
			if err := validateCompositionArray(x, childPath, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSuperString(s string, context markerContext) error {
	if !strings.HasPrefix(s, superToken) {
		return nil
	}
	rest := strings.TrimPrefix(s, superToken)
	if rest != "" {
		first, _ := utf8FirstRune(rest)
		if isIdentifierRune(first) {
			return nil
		}
		return fmt.Errorf("unknown marker %q (unknown token): the %q namespace is reserved for future syntax", s, superToken)
	}
	if context == markerArrayElement {
		return fmt.Errorf("unresolved marker %q: no inherited array to compose with", s)
	}
	return fmt.Errorf("marker %q is out of context", s)
}

func utf8FirstRune(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return 0, 0
}

func isIdentifierRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func jsonTypeName(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case bool:
		return "boolean"
	default:
		return "number"
	}
}

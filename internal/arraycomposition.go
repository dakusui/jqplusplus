package internal

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	superToken      = "$super"
	superMergeToken = "$super*"
)

// containsSuperToken reports whether arr contains either array-composition
// marker as a direct element. Markers in an inner array have no composition
// meaning.
func containsSuperToken(arr []any) bool {
	for _, element := range arr {
		if s, ok := element.(string); ok && (s == superToken || s == superMergeToken) {
			return true
		}
	}
	return false
}

// composeSuperArray resolves the child's direct marker against parent. A
// splice delta can compose with another splice delta, but pairing deltas cannot
// compose with either marker because their flat syntax cannot preserve the
// pairing delimiter's meaning across a second merge.
func composeSuperArray(parent, child []any, path string) ([]any, error) {
	nSplices, mergeAt := 0, -1
	for i, element := range child {
		s, ok := element.(string)
		if !ok {
			continue
		}
		switch s {
		case superToken:
			nSplices++
		case superMergeToken:
			if mergeAt >= 0 {
				return nil, fmt.Errorf("%q may appear at most once in an array", superMergeToken)
			}
			mergeAt = i
		}
	}
	if nSplices > 0 && mergeAt >= 0 {
		return nil, fmt.Errorf("%q and %q cannot be mixed in one array", superToken, superMergeToken)
	}
	if mergeAt >= 0 {
		if containsSuperToken(parent) {
			return nil, fmt.Errorf("cannot compose a pairing delta with an inherited marked array")
		}
		return pairSuperArray(parent, child, mergeAt, path)
	}
	if containsSuperMergeToken(parent) {
		return nil, fmt.Errorf("cannot compose a splice delta with an inherited pairing delta")
	}
	return spliceSuperArray(parent, child), nil
}

func containsSuperMergeToken(arr []any) bool {
	for _, element := range arr {
		if s, ok := element.(string); ok && s == superMergeToken {
			return true
		}
	}
	return false
}

// spliceSuperArray substitutes the parent elements at every splice marker. It
// always allocates a new slice so that a cached inherited array is never
// modified through the result.
func spliceSuperArray(parent, child []any) []any {
	result := make([]any, 0, len(parent)+len(child))
	for _, element := range child {
		if element == superToken {
			result = append(result, parent...)
			continue
		}
		result = append(result, element)
	}
	return result
}

func pairSuperArray(parent, child []any, mergeAt int, path string) ([]any, error) {
	prefix, queue := child[:mergeAt], child[mergeAt+1:]
	result := make([]any, 0, len(prefix)+len(parent)+len(queue))
	result = append(result, prefix...)
	for i := 0; i < len(parent) || i < len(queue); i++ {
		switch {
		case i < len(parent) && i < len(queue):
			merged, err := mergePairedValues(parent[i], queue[i], fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			result = append(result, merged)
		case i < len(parent):
			result = append(result, parent[i])
		default:
			result = append(result, queue[i])
		}
	}
	return result, nil
}

func mergePairedValues(parent, child any, path string) (any, error) {
	parentKind, childKind := kindOf(parent), kindOf(child)
	if parentKind != childKind {
		return nil, fmt.Errorf("at %s: cross-kind pair (%s with %s): a merge is only defined between values of the same kind", path, parentKind, childKind)
	}
	switch parentKind {
	case kindObject:
		return mergeObjectsAtPath(parent.(map[string]any), child.(map[string]any), MergePolicyDefault, path)
	case kindArray:
		childArray := child.([]any)
		if containsSuperToken(childArray) {
			return composeSuperArray(parent.([]any), childArray, path)
		}
		return childArray, nil
	default:
		return child, nil
	}
}

const (
	kindObject = "object"
	kindArray  = "array"
	kindAtom   = "atom"
)

func kindOf(value any) string {
	switch value.(type) {
	case map[string]any:
		return kindObject
	case []any:
		return kindArray
	default:
		return kindAtom
	}
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
	if directObjectArray {
		nSplices, nPairings := 0, 0
		for _, value := range arr {
			s, ok := value.(string)
			if !ok {
				continue
			}
			if err := validateSuperString(s, markerArrayElement); err != nil {
				return err
			}
			switch s {
			case superToken:
				nSplices++
			case superMergeToken:
				nPairings++
			}
		}
		if nPairings > 1 {
			return fmt.Errorf("%q may appear at most once in an array", superMergeToken)
		}
		if nSplices > 0 && nPairings > 0 {
			return fmt.Errorf("%q and %q cannot be mixed in one array", superToken, superMergeToken)
		}
		if nSplices > 0 || nPairings > 0 {
			return fmt.Errorf("unresolved marker: unresolved composition tokens have no inherited array to compose with")
		}
	}
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
	if s == superToken || s == superMergeToken {
		if context == markerArrayElement {
			return nil
		}
		return fmt.Errorf("marker %q is out of context", s)
	}
	if rest != "" {
		first, _ := utf8FirstRune(rest)
		if isIdentifierRune(first) {
			return nil
		}
		return fmt.Errorf("unknown marker %q (unknown token): the %q namespace is reserved for future syntax", s, superToken)
	}
	panic("unreachable")
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

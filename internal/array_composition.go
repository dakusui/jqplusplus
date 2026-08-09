package internal

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const (
	superSpliceToken = "$super"
	superMergeToken  = "$super*"
)

type arrayComposition int

const (
	arrayCompositionNone arrayComposition = iota
	arrayCompositionSplice
	arrayCompositionMerge
)

type jsonValueKind string

const (
	jsonKindObject jsonValueKind = "object"
	jsonKindArray  jsonValueKind = "array"
	jsonKindAtom   jsonValueKind = "atom"
)

func mergeValues(inherited, child any, policy MergePolicy) (any, error) {
	if inheritedObject, ok := inherited.(map[string]any); ok {
		if childObject, ok := child.(map[string]any); ok {
			return MergeObjects(inheritedObject, childObject, policy)
		}
	}

	childArray, childIsArray := child.([]any)
	if !childIsArray {
		return child, nil
	}
	composition, err := inspectArrayComposition(childArray)
	if err != nil {
		return nil, err
	}
	if composition == arrayCompositionNone {
		return child, nil
	}

	inheritedArray, ok := inherited.([]any)
	if !ok {
		return nil, fmt.Errorf("cannot apply %q to inherited %s; expected array", composition.token(), kindOf(inherited))
	}
	parentComposition, err := inspectArrayComposition(inheritedArray)
	if err != nil {
		return nil, fmt.Errorf("invalid inherited array delta: %w", err)
	}
	if (composition == arrayCompositionMerge && parentComposition != arrayCompositionNone) ||
		(composition == arrayCompositionSplice && parentComposition == arrayCompositionMerge) {
		return nil, fmt.Errorf("cannot compose marked arrays %q and %q", parentComposition.token(), composition.token())
	}

	switch composition {
	case arrayCompositionSplice:
		return spliceInheritedArray(inheritedArray, childArray), nil
	case arrayCompositionMerge:
		return mergeInheritedArray(inheritedArray, childArray, policy)
	default:
		panic("unexpected array composition")
	}
}

func inspectArrayComposition(array []any) (arrayComposition, error) {
	spliceCount := 0
	mergeCount := 0
	for _, element := range array {
		token, ok := element.(string)
		if !ok {
			continue
		}
		switch token {
		case superSpliceToken:
			spliceCount++
		case superMergeToken:
			mergeCount++
		}
	}
	if spliceCount > 0 && mergeCount > 0 {
		return arrayCompositionNone, fmt.Errorf("%q and %q cannot be used in the same array", superSpliceToken, superMergeToken)
	}
	if mergeCount > 1 {
		return arrayCompositionNone, fmt.Errorf("%q may appear only once in an array", superMergeToken)
	}
	if mergeCount == 1 {
		return arrayCompositionMerge, nil
	}
	if spliceCount > 0 {
		return arrayCompositionSplice, nil
	}
	return arrayCompositionNone, nil
}

func (c arrayComposition) token() string {
	switch c {
	case arrayCompositionSplice:
		return superSpliceToken
	case arrayCompositionMerge:
		return superMergeToken
	default:
		return "unmarked"
	}
}

func spliceInheritedArray(inherited, child []any) []any {
	spliceCount := 0
	for _, element := range child {
		if token, ok := element.(string); ok && token == superSpliceToken {
			spliceCount++
		}
	}
	result := make([]any, 0, len(child)-spliceCount+spliceCount*len(inherited))
	for _, element := range child {
		if token, ok := element.(string); ok && token == superSpliceToken {
			result = append(result, inherited...)
			continue
		}
		result = append(result, element)
	}
	return result
}

func mergeInheritedArray(inherited, child []any, policy MergePolicy) ([]any, error) {
	marker := -1
	for i, element := range child {
		if token, ok := element.(string); ok && token == superMergeToken {
			marker = i
			break
		}
	}

	prefix := child[:marker]
	queue := child[marker+1:]
	paired := len(inherited)
	if len(queue) < paired {
		paired = len(queue)
	}
	result := make([]any, 0, len(prefix)+max(len(inherited), len(queue)))
	result = append(result, prefix...)
	for i := 0; i < paired; i++ {
		merged, err := mergeArrayPair(inherited[i], queue[i], policy)
		if err != nil {
			return nil, fmt.Errorf("cannot merge array elements at index %d: %w", i, err)
		}
		result = append(result, merged)
	}
	result = append(result, inherited[paired:]...)
	result = append(result, queue[paired:]...)
	return result, nil
}

func mergeArrayPair(inherited, child any, policy MergePolicy) (any, error) {
	inheritedKind := kindOf(inherited)
	childKind := kindOf(child)
	if inheritedKind != childKind {
		return nil, fmt.Errorf("cross-kind merge is not allowed: inherited %s, child %s", inheritedKind, childKind)
	}
	switch inheritedKind {
	case jsonKindObject:
		return MergeObjects(inherited.(map[string]any), child.(map[string]any), policy)
	case jsonKindArray:
		return mergeValues(inherited, child, policy)
	case jsonKindAtom:
		return child, nil
	default:
		panic("unexpected JSON value kind")
	}
}

func kindOf(value any) jsonValueKind {
	switch value.(type) {
	case map[string]any:
		return jsonKindObject
	case []any:
		return jsonKindArray
	default:
		return jsonKindAtom
	}
}

// GroundArrayCompositions verifies that no array-composition intent remains
// after all file- and node-level inheritance has been resolved. It also
// enforces the reserved $super token-family namespace.
func GroundArrayCompositions(obj map[string]any) error {
	return groundArrayCompositionsAt(obj, nil)
}

func groundArrayCompositionsAt(value any, path []any) error {
	switch value := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if err := groundArrayCompositionsAt(value[key], appendPath(path, key)); err != nil {
				return err
			}
		}
	case []any:
		if _, err := inspectArrayComposition(value); err != nil {
			return fmt.Errorf("invalid array composition at %s: %w", displayPath(path), err)
		}
		for i, element := range value {
			if token, ok := element.(string); ok {
				if isUnknownSuperToken(token) {
					return fmt.Errorf("unknown array-composition token %q at %s", token, displayPath(appendPath(path, i)))
				}
				if token == superSpliceToken || token == superMergeToken {
					return fmt.Errorf("unresolved array-composition token %q at %s", token, displayPath(appendPath(path, i)))
				}
			}
			if err := groundArrayCompositionsAt(element, appendPath(path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func isUnknownSuperToken(value string) bool {
	if !strings.HasPrefix(value, superSpliceToken) || value == superSpliceToken || value == superMergeToken {
		return false
	}
	remainder := strings.TrimPrefix(value, superSpliceToken)
	first := firstRune(remainder)
	return !isIdentifierRune(first)
}

func firstRune(value string) rune {
	for _, r := range value {
		return r
	}
	return 0
}

func isIdentifierRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func appendPath(path []any, element any) []any {
	result := make([]any, len(path), len(path)+1)
	copy(result, path)
	return append(result, element)
}

func displayPath(path []any) string {
	expression, err := PathArrayToPathExpression(path)
	if err != nil || expression == "" {
		return "."
	}
	return expression
}

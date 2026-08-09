package internal

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const (
	superToken      = "$super"
	superMergeToken = "$super*"
)

type arrayCompositionKind int

const (
	arrayCompositionNone arrayCompositionKind = iota
	arrayCompositionSplice
	arrayCompositionPairwise
)

// arrayCompositionMarker describes direct composition tokens in an array.
// Tokens nested inside a value are intentionally not considered here; that
// nested array is considered only if it is itself merged.
func arrayCompositionMarker(array []any) (arrayCompositionKind, error) {
	superCount := 0
	superMergeCount := 0
	for _, value := range array {
		stringValue, ok := value.(string)
		if !ok {
			continue
		}
		switch stringValue {
		case superToken:
			superCount++
		case superMergeToken:
			superMergeCount++
		}
	}
	if superCount > 0 && superMergeCount > 0 {
		return arrayCompositionNone, fmt.Errorf("cannot mix %s and %s in the same array", superToken, superMergeToken)
	}
	if superMergeCount > 1 {
		return arrayCompositionNone, fmt.Errorf("only one %s is allowed in an array", superMergeToken)
	}
	if superCount > 0 {
		return arrayCompositionSplice, nil
	}
	if superMergeCount == 1 {
		return arrayCompositionPairwise, nil
	}
	return arrayCompositionNone, nil
}

func mergeArrayComposition(parent any, child []any, policy MergePolicy) ([]any, error) {
	childMarker, err := arrayCompositionMarker(child)
	if err != nil {
		return nil, err
	}
	if childMarker == arrayCompositionNone {
		// Array replacement remains the default inheritance behaviour.
		return child, nil
	}

	parentArray, parentIsArray := parent.([]any)
	if !parentIsArray {
		return nil, fmt.Errorf("%s requires an inherited array, but the inherited value is %s", compositionTokenName(childMarker), jsonValueKind(parent))
	}
	parentMarker, err := arrayCompositionMarker(parentArray)
	if err != nil {
		return nil, err
	}
	if parentMarker != arrayCompositionNone {
		return nil, fmt.Errorf("cannot merge two marked arrays (%s and %s)", compositionTokenName(parentMarker), compositionTokenName(childMarker))
	}

	switch childMarker {
	case arrayCompositionSplice:
		return spliceInheritedArray(parentArray, child), nil
	case arrayCompositionPairwise:
		return pairwiseMergeInheritedArray(parentArray, child, policy)
	default:
		panic("unknown array composition kind")
	}
}

func compositionTokenName(kind arrayCompositionKind) string {
	switch kind {
	case arrayCompositionSplice:
		return superToken
	case arrayCompositionPairwise:
		return superMergeToken
	default:
		return "array composition token"
	}
}

func spliceInheritedArray(parent, child []any) []any {
	tokenCount := 0
	for _, value := range child {
		if value == superToken {
			tokenCount++
		}
	}
	result := make([]any, 0, len(child)-tokenCount+len(parent)*tokenCount)
	for _, value := range child {
		if value == superToken {
			result = append(result, parent...)
			continue
		}
		result = append(result, value)
	}
	return result
}

func pairwiseMergeInheritedArray(parent, child []any, policy MergePolicy) ([]any, error) {
	markerIndex := -1
	for i, value := range child {
		if value == superMergeToken {
			markerIndex = i
			break
		}
	}
	prefix := child[:markerIndex]
	queue := child[markerIndex+1:]
	pairedCount := len(parent)
	if len(queue) < pairedCount {
		pairedCount = len(queue)
	}

	result := make([]any, 0, len(prefix)+len(parent)+len(queue)-pairedCount)
	result = append(result, prefix...)
	for i := 0; i < pairedCount; i++ {
		merged, err := mergePairwiseValues(parent[i], queue[i], policy)
		if err != nil {
			return nil, fmt.Errorf("at index %d: %w", i, err)
		}
		result = append(result, merged)
	}
	result = append(result, parent[pairedCount:]...)
	result = append(result, queue[pairedCount:]...)
	return result, nil
}

func mergePairwiseValues(parent, child any, policy MergePolicy) (any, error) {
	parentKind := jsonValueKind(parent)
	childKind := jsonValueKind(child)
	if parentKind != childKind {
		return nil, fmt.Errorf("cannot pairwise merge %s with %s", parentKind, childKind)
	}
	return mergeValues(parent, child, policy)
}

type jsonKind int

const (
	jsonAtom jsonKind = iota
	jsonArray
	jsonObject
)

func (kind jsonKind) String() string {
	switch kind {
	case jsonObject:
		return "object"
	case jsonArray:
		return "array"
	default:
		return "atom"
	}
}

func jsonValueKind(value any) jsonKind {
	if _, ok := value.(map[string]any); ok {
		return jsonObject
	}
	if _, ok := value.([]any); ok {
		return jsonArray
	}
	return jsonAtom
}

// ValidateArrayComposition grounds inherited-array deltas after all file- and
// node-level inheritance has completed. It also reserves the $super token
// family so future spellings cannot silently become literal values today.
func ValidateArrayComposition(root map[string]any) error {
	return validateArrayCompositionValue(root, nil)
}

func validateArrayCompositionValue(value any, path []any) error {
	switch typedValue := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typedValue))
		for key := range typedValue {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if err := validateArrayCompositionValue(typedValue[key], appendPath(path, key)); err != nil {
				return err
			}
		}
	case []any:
		if _, err := arrayCompositionMarker(typedValue); err != nil {
			return fmt.Errorf("%w at %s", err, pathExpression(path))
		}
		for index, element := range typedValue {
			if tokenError := validateArrayToken(element, appendPath(path, index)); tokenError != nil {
				return tokenError
			}
			if err := validateArrayCompositionValue(element, appendPath(path, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func appendPath(path []any, element any) []any {
	result := make([]any, len(path), len(path)+1)
	copy(result, path)
	return append(result, element)
}

func validateArrayToken(value any, path []any) error {
	stringValue, ok := value.(string)
	if !ok {
		return nil
	}
	switch classifySuperToken(stringValue) {
	case arrayCompositionSplice, arrayCompositionPairwise:
		return fmt.Errorf("unresolved array composition token %q at %s", stringValue, pathExpression(path))
	case arrayCompositionNone:
		return nil
	default:
		return fmt.Errorf("unknown array composition token %q at %s", stringValue, pathExpression(path))
	}
}

const unknownArrayCompositionToken arrayCompositionKind = -1

func classifySuperToken(value string) arrayCompositionKind {
	switch value {
	case superToken:
		return arrayCompositionSplice
	case superMergeToken:
		return arrayCompositionPairwise
	}
	if !strings.HasPrefix(value, superToken) {
		return arrayCompositionNone
	}
	suffix := strings.TrimPrefix(value, superToken)
	firstRune, _ := utf8FirstRune(suffix)
	if isIdentifierRune(firstRune) {
		return arrayCompositionNone
	}
	return unknownArrayCompositionToken
}

func utf8FirstRune(value string) (rune, bool) {
	for _, r := range value {
		return r, true
	}
	return 0, false
}

func isIdentifierRune(value rune) bool {
	return value == '_' || unicode.IsLetter(value) || unicode.IsDigit(value)
}

func pathExpression(path []any) string {
	if len(path) == 0 {
		return "."
	}
	expression, err := PathArrayToPathExpression(path)
	if err != nil {
		return fmt.Sprint(path)
	}
	return expression
}

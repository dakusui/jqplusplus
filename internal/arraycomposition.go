package internal

import (
	"fmt"
	"maps"
	"sort"
	"strings"
	"unicode"
)

// Array composition (issue #74).
//
// During inheritance merge ($extends / $includes, file-level and node-level),
// two tokens are recognized as direct string elements of an array:
//
//   "$super"  — splice: the inherited array's elements replace the token, verbatim.
//   "$super*" — pairwise merge: elements before the token are a literal prefix;
//               elements after the token are merged index-wise with the
//               inherited elements.
//
// An array containing a token is a delta. When the inherited value for the
// same key is a plain (unmarked) array, the delta is grounded against it.
// When the key is absent from the inherited object, the delta carries
// unchanged; a delta still unresolved after all inheritance is settled is a
// configuration error, reported by ValidateArrayCompositions. Unmarked arrays
// keep the default behavior: the child array replaces the inherited one.

const spliceToken = "$super"
const pairwiseToken = "$super*"

// arrayMarking classifies how an array participates in array composition.
type arrayMarking int

const (
	unmarked arrayMarking = iota
	spliceMarked
	pairwiseMarked
)

// MergeNodes merges an inherited (parent) object with an inheriting (child)
// object, with the child's values taking precedence. It is the merge entry
// point used by the $extends / $includes resolution and, unlike MergeObjects,
// implements array composition and therefore can fail with a configuration
// error.
func MergeNodes(parent, child map[string]any) (map[string]any, error) {
	return mergeNodeObjects(parent, child, nil)
}

func mergeNodeObjects(parent, child map[string]any, path []any) (map[string]any, error) {
	result := make(map[string]any, len(parent)+len(child))
	maps.Copy(result, parent)
	for k, v := range child {
		parentValue, parentExists := result[k]
		merged, err := mergeNodeValues(parentValue, parentExists, v, append(path, k))
		if err != nil {
			return nil, err
		}
		result[k] = merged
	}
	return result, nil
}

// mergeNodeValues combines an inherited value with a child value at one key.
// Objects merge recursively; a marked child array composes with the inherited
// value; everything else is an override where the child wins. parentExists
// distinguishes an absent inherited key (a delta carries) from an inherited
// null (an atom, so a marked child array is a configuration error).
func mergeNodeValues(parent any, parentExists bool, child any, path []any) (any, error) {
	if po, ok := parent.(map[string]any); ok {
		if co, ok := child.(map[string]any); ok {
			return mergeNodeObjects(po, co, path)
		}
	}
	if ca, ok := child.([]any); ok {
		marking, err := inspectArrayMarking(ca, path)
		if err != nil {
			return nil, err
		}
		if marking == unmarked {
			return child, nil
		}
		if !parentExists {
			// Key absent on the inherited side: the delta carries unchanged.
			return child, nil
		}
		pa, ok := parent.([]any)
		if !ok {
			return nil, fmt.Errorf("array composition: %q requires the inherited value to be an array, but got %s: at '%s'", tokenOf(marking), kindOf(parent), toPathExpression(path))
		}
		parentMarking, err := inspectArrayMarking(pa, path)
		if err != nil {
			return nil, err
		}
		switch marking {
		case spliceMarked:
			if parentMarking == pairwiseMarked {
				return nil, fmt.Errorf("array composition: cannot splice into an inherited pairwise delta: at '%s'", toPathExpression(path))
			}
			// An inherited splice delta composes: its own tokens are carried
			// into the result as ordinary elements and stay unresolved.
			return spliceArrays(pa, ca), nil
		case pairwiseMarked:
			if parentMarking != unmarked {
				return nil, fmt.Errorf("array composition: cannot pairwise-merge with an inherited marked array: at '%s'", toPathExpression(path))
			}
			return pairwiseMergeArrays(pa, ca, path)
		}
	}
	return child, nil
}

// inspectArrayMarking classifies an array's participation in array
// composition, looking only at its direct string elements. Tokens inside
// nested arrays are inert here; they are either resolved when the nested
// array itself is composed, or reported by ValidateArrayCompositions.
func inspectArrayMarking(arr []any, path []any) (arrayMarking, error) {
	splices, pairwises := 0, 0
	for _, e := range arr {
		switch e {
		case spliceToken:
			splices++
		case pairwiseToken:
			pairwises++
		}
	}
	if pairwises > 0 && splices > 0 {
		return unmarked, fmt.Errorf("array composition: %q and %q cannot appear in the same array: at '%s'", spliceToken, pairwiseToken, toPathExpression(path))
	}
	if pairwises > 1 {
		return unmarked, fmt.Errorf("array composition: at most one %q token is allowed in an array: at '%s'", pairwiseToken, toPathExpression(path))
	}
	switch {
	case splices > 0:
		return spliceMarked, nil
	case pairwises > 0:
		return pairwiseMarked, nil
	default:
		return unmarked, nil
	}
}

func tokenOf(m arrayMarking) string {
	if m == pairwiseMarked {
		return pairwiseToken
	}
	return spliceToken
}

// spliceArrays replaces every splice token in child with the elements of
// parent, verbatim. It always builds a fresh slice; inputs are never mutated,
// so cached inherited arrays stay intact.
func spliceArrays(parent, child []any) []any {
	result := make([]any, 0, len(child)+len(parent))
	for _, e := range child {
		if e == spliceToken {
			result = append(result, parent...)
			continue
		}
		result = append(result, e)
	}
	return result
}

// pairwiseMergeArrays grounds a pairwise delta (child) against a plain
// inherited array (parent). Child elements before the token are a literal
// prefix; elements after it are merged index-wise with parent's elements.
// Unpaired elements survive on both sides: parent's extras follow the merged
// region, and child's extras become a literal tail.
func pairwiseMergeArrays(parent, child []any, path []any) ([]any, error) {
	tokenIndex := 0
	for i, e := range child {
		if e == pairwiseToken {
			tokenIndex = i
			break
		}
	}
	prefix := child[:tokenIndex]
	queue := child[tokenIndex+1:]

	result := make([]any, 0, len(prefix)+len(parent)+len(queue))
	result = append(result, prefix...)
	for i := 0; i < len(parent) || i < len(queue); i++ {
		switch {
		case i < len(parent) && i < len(queue):
			merged, err := mergePairedValues(parent[i], queue[i], append(path, len(result)))
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

// mergePairedValues merges one inherited element with the child element paired
// to it by a pairwise merge, following the same-kind merge principle:
// object×object merges recursively, array×array recurses into array
// composition (the child replaces unless it is marked), atom×atom lets the
// child win, and any cross-kind pair is a configuration error.
func mergePairedValues(parent, child any, path []any) (any, error) {
	pk, ck := kindOf(parent), kindOf(child)
	if pk != ck {
		return nil, fmt.Errorf("array composition: cannot merge %s with %s in a pairwise merge (same-kind merge principle): at '%s'", pk, ck, toPathExpression(path))
	}
	switch ck {
	case "object":
		return mergeNodeObjects(parent.(map[string]any), child.(map[string]any), path)
	case "array":
		return mergeNodeValues(parent, true, child, path)
	default:
		return child, nil
	}
}

func kindOf(v any) string {
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return "atom"
	}
}

// ValidateArrayCompositions is the final validation sweep, run on the
// top-level object after all inheritance is resolved and before key-side
// evaluation. It walks the whole object deterministically (object keys in
// sorted order) and reports the first offending array string element:
//
//  1. a remaining "$super" / "$super*" token is dangling merge intent — it can
//     never be resolved anymore;
//  2. any other string extending "$super" with a non-identifier character
//     (such as "$super?" or "$super*(name)") belongs to the reserved token
//     namespace and is an unknown token;
//  3. "$super" followed by an identifier character (such as "$supervisor") is
//     an ordinary literal and is left untouched.
func ValidateArrayCompositions(obj map[string]any) error {
	return validateComposedValue(obj, nil)
}

func validateComposedValue(v any, path []any) error {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err := validateComposedValue(x[k], append(path, k)); err != nil {
				return err
			}
		}
	case []any:
		for i, e := range x {
			if s, ok := e.(string); ok {
				if err := classifyArrayStringElement(s, append(path, i)); err != nil {
					return err
				}
				continue
			}
			if err := validateComposedValue(e, append(path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func classifyArrayStringElement(s string, path []any) error {
	if s == spliceToken || s == pairwiseToken {
		return fmt.Errorf("array composition: unresolved token %q: no inherited array to compose with: at '%s'", s, toPathExpression(path))
	}
	rest, ok := strings.CutPrefix(s, spliceToken)
	if !ok || rest == "" {
		return nil
	}
	r := []rune(rest)[0]
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		// An ordinary literal such as "$supervisor"; future token-family
		// members only ever extend "$super" with non-identifier characters.
		return nil
	}
	return fmt.Errorf("array composition: unknown token %q: the %q token namespace is reserved: at '%s'", s, spliceToken, toPathExpression(path))
}

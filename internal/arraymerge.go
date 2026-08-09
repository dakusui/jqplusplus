package internal

import (
	"fmt"
	"strings"
)

// Array-composition tokens (issue #74).
//
// Both tokens are recognized only as direct string elements of an array that
// participates in inheritance merging ($extends / $includes). An unmarked
// child array keeps the pre-existing override semantics: it replaces the
// inherited array wholesale.
const (
	// SuperToken splices the inherited array's elements, verbatim, at the
	// position where it appears. It may appear multiple times.
	SuperToken = "$super"
	// SuperMergeToken switches from literal prefix to index-wise pairwise
	// merging with the inherited elements. It may appear at most once.
	SuperMergeToken = "$super*"
)

// isSuperFamilyToken reports whether s is one of the recognized composition tokens.
func isSuperFamilyToken(s string) bool {
	return s == SuperToken || s == SuperMergeToken
}

// checkSuperFamilyString classifies s against the reserved "$super" name space.
// Strings where "$super" is followed by an identifier character (e.g.
// "$supervisor") are ordinary literals — they are different words. Anything
// else that begins with "$super" must be an exact known token; otherwise it is
// an unknown-token error, keeping the name space free for future syntax
// (e.g. a lenient "$super?") to be added without changing the meaning of any
// currently valid configuration.
func checkSuperFamilyString(s string) error {
	if isSuperFamilyToken(s) || !strings.HasPrefix(s, SuperToken) {
		return nil
	}
	rest := s[len(SuperToken):]
	if rest == "" || isIdentifierChar(rest[0]) {
		return nil
	}
	return fmt.Errorf("unknown token %q: only %q and %q are recognized; the %q name space is reserved for future syntax (escape a literal with \"raw:\")", s, SuperToken, SuperMergeToken, SuperToken)
}

func isIdentifierChar(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// containsSuperToken reports whether arr has a composition token among its direct elements.
func containsSuperToken(arr []any) bool {
	for _, e := range arr {
		if s, ok := e.(string); ok && isSuperFamilyToken(s) {
			return true
		}
	}
	return false
}

// composeArrays combines a child array containing composition tokens with the
// inherited parent array. It always returns a freshly allocated slice: parent
// slices may be shared with the node-pool cache and must never be mutated.
func composeArrays(parent, child []any, path string) ([]any, error) {
	nSplice, mergeAt := 0, -1
	for i, e := range child {
		s, ok := e.(string)
		if !ok {
			continue
		}
		switch s {
		case SuperToken:
			nSplice++
		case SuperMergeToken:
			if mergeAt >= 0 {
				return nil, fmt.Errorf("at %s: %q may appear at most once in an array", path, SuperMergeToken)
			}
			mergeAt = i
		}
	}
	if mergeAt >= 0 && nSplice > 0 {
		return nil, fmt.Errorf("at %s: %q and %q cannot be mixed in one array", path, SuperToken, SuperMergeToken)
	}
	if mergeAt >= 0 {
		return pairwiseMergeArrays(parent, child, mergeAt, path)
	}
	// Splice: expand each SuperToken occurrence to the parent's elements.
	// Parent elements are copied verbatim; composition tokens the parent may
	// itself carry stay in place, which is what lets deltas compose across
	// inheritance chains before they are finally grounded.
	result := make([]any, 0, len(child)+len(parent))
	for _, e := range child {
		if s, ok := e.(string); ok && s == SuperToken {
			result = append(result, parent...)
			continue
		}
		result = append(result, e)
	}
	return result, nil
}

// pairwiseMergeArrays implements the SuperMergeToken semantics: elements before
// the token are a literal prefix; elements after it (the queue) merge
// index-wise with the inherited elements. Unpaired extras survive on both
// sides: the parent's follow the merged region in order, the queue's become a
// literal tail.
func pairwiseMergeArrays(parent, child []any, mergeAt int, path string) ([]any, error) {
	if containsSuperToken(parent) {
		return nil, fmt.Errorf("at %s: cannot pairwise-merge with an inherited array that itself contains unresolved composition tokens", path)
	}
	prefix, queue := child[:mergeAt], child[mergeAt+1:]
	result := make([]any, 0, len(prefix)+len(parent)+len(queue))
	result = append(result, prefix...)
	for i := 0; i < len(parent) || i < len(queue); i++ {
		switch {
		case i < len(parent) && i < len(queue):
			v, err := mergePairValues(parent[i], queue[i], fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			result = append(result, v)
		case i < len(parent):
			result = append(result, parent[i])
		default:
			result = append(result, queue[i])
		}
	}
	return result, nil
}

// mergePairValues merges one paired element under the same-kind merge
// principle: object×object merges deeply (with ordinary override semantics at
// its keys), array×array follows the array-composition rules, atom×atom takes
// the child value, and any cross-kind pair is a configuration error.
func mergePairValues(parent, child any, path string) (any, error) {
	pk, ck := kindClass(parent), kindClass(child)
	if pk != ck {
		return nil, fmt.Errorf("at %s: cannot merge %s with %s: a merge is only defined between values of the same kind", path, jsonTypeName(parent), jsonTypeName(child))
	}
	switch pk {
	case kindObject:
		return mergeObjectsAtPath(parent.(map[string]any), child.(map[string]any), MergePolicyDefault, path)
	case kindArray:
		cv := child.([]any)
		if containsSuperToken(cv) {
			return composeArrays(parent.([]any), cv, path)
		}
		return cv, nil
	default:
		return child, nil
	}
}

const (
	kindObject = "object"
	kindArray  = "array"
	kindAtom   = "atom"
)

func kindClass(v any) string {
	switch v.(type) {
	case map[string]any:
		return kindObject
	case []any:
		return kindArray
	default:
		return kindAtom
	}
}

func jsonTypeName(v any) string {
	switch v.(type) {
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

// ValidateArrayComposition walks a fully composed object and reports
// composition tokens that can no longer be resolved (dangling merge intent) as
// well as unknown tokens in the reserved "$super" name space. It must run only
// on the top-level object — after inheritance resolution, before evaluation —
// so that unresolved deltas inside parent fragments can still compose, while
// nothing unresolved ever reaches the output.
func ValidateArrayComposition(obj map[string]any) error {
	return validateComposedValue(obj, "")
}

func validateComposedValue(v any, path string) error {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			if err := validateComposedValue(e, path+"."+k); err != nil {
				return err
			}
		}
	case []any:
		for i, e := range x {
			p := fmt.Sprintf("%s[%d]", path, i)
			if s, ok := e.(string); ok {
				if isSuperFamilyToken(s) {
					return fmt.Errorf("unresolved %q at %s: no inherited array to compose with (escape a literal with \"raw:\")", s, p)
				}
				if err := checkSuperFamilyString(s); err != nil {
					return fmt.Errorf("%v: at %s", err, p)
				}
			}
			if err := validateComposedValue(e, p); err != nil {
				return err
			}
		}
	}
	return nil
}

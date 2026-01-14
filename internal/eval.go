package internal

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/itchyny/gojq"
)

type JSONType int

const (
	Null = iota
	Bool
	String
	Number
	Array
	Object
)

func (t JSONType) String() string {
	switch t {
	case Null:
		return "null"
	case Bool:
		return "bool"
	case String:
		return "string"
	case Number:
		return "number"
	case Array:
		return "array"
	case Object:
		return "object"
	default:
		return "unknown"
	}
}

// ApplyJQExpression applies a jq expression to the provided input object, validates the result type,
// and returns it in the specified type.
// ApplyJQExpression applies a jq expression to the provided input object, validates the result type,
// and returns it in the specified type.
//
// NOTE: Custom jq functions/modules are enabled by compiling the parsed query with compiler options
// (e.g., gojq.WithFunction, gojq.WithModuleLoader, ...).
func ApplyJQExpression(
	input any,
	expression string,
	expectedTypes []JSONType,
	compilerOpts ...gojq.CompilerOption,
) (any, error) {
	// Parse the jq expression
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jq expression: %w", err)
	}

	// Compile the jq query (this is where custom functions/modules are wired in)
	code, err := gojq.Compile(query, compilerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to compile jq expression: %w", err)
	}

	// Run the compiled jq code
	iter := code.Run(input)

	result, ok := iter.Next()
	if !ok {
		return nil, fmt.Errorf("no result produced by jq expression")
	}

	// Check if the result is an error
	if err, isErr := result.(error); isErr {
		return nil, fmt.Errorf("error while executing jq expression: %w", err)
	}

	// Validate and return the result based on the expected type
	expected := isExpected(result, expectedTypes...)
	if !expected {
		return nil, fmt.Errorf("result type mismatch: expected one of %s but got %T", expectedTypes, result)
	}
	return result, nil
}

func isExpected(v any, expectedTypes ...JSONType) bool {
	for _, each := range expectedTypes {
		switch each {
		case String:
			if _, ok := v.(string); ok {
				return true
			}
		case Array:
			k := reflect.TypeOf(v).Kind()
			if k == reflect.Slice || k == reflect.Array {
				return true
			}
		case Object:
			if _, ok := v.(map[string]any); ok {
				return true
			}
		case Number:
			switch v.(type) {
			case float64:
				return true
			case int:
				return true
			case int64:
				return true
			default:
				continue
			}
		case Bool:
			if _, ok := v.(bool); ok {
				return true
			}
		case Null:
			if v == nil {
				return true
			}
		default:
			continue
		}
	}
	return false
}

func toStringArray(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{v.(string)}
	case []string:
		return v.([]string)
	default:
		panic(fmt.Sprintf("Unexpected type: %v (%v)", x, v))
	}
}

func ProcessKeySide(obj map[string]any, ttl int) (map[string]any, error) {
	keyHavingPrefixForProcessing := func(path []any) bool {
		last := path[len(path)-1]
		switch last.(type) {
		case string:
			break
		default:
			return false
		}
		key := last.(string)
		return strings.HasPrefix(key, "eval:") || strings.HasPrefix(key, "raw:")
	}
	type keyChange struct {
		// The last element must be a string
		Before []any
		// An array each of which should replace the last element of Before.
		After []string
	}
	// Process keys
	pathsToBeProcessed := Paths(obj, keyHavingPrefixForProcessing)
	if len(pathsToBeProcessed) == 0 {
		return obj, nil
	}
	if ttl <= 0 {
		panic(fmt.Sprintf("ttl is 0, %v entries left.(%v)", len(pathsToBeProcessed), pathsToBeProcessed))
	}
	keyChanges := Map(pathsToBeProcessed, func(p []any) keyChange {
		str := p[len(p)-1]
		if strings.HasPrefix(str.(string), "raw:") {
			return keyChange{
				Before: p,
				After:  []string{str.(string)[len("raw:"):]},
			}
		} else if strings.HasPrefix(str.(string), "eval:") {
			expr, t := extractExpressionAndExpectedType(str.(string)[len("eval:"):])
			if t != String && t != Array {
				panic(fmt.Sprintf("Last element of path must be a string or an array: %v", p))
			}
			v, err := ApplyJQExpression(obj, expr, []JSONType{String, Array})
			if err != nil {
				panic(fmt.Sprintf("Failed to evaluate jq expression: %v", err))
			}
			w := toStringArray(v)
			ret := keyChange{
				Before: p,
				After:  w,
			}
			return ret
		}
		panic(fmt.Sprintf("Last element of path must start with eval: or raw: %v", p))
	})
	ret := DeepCopyAs(obj)
	for _, c := range keyChanges {
		var v any
		v, ok := GetAtPath(ret, c.Before)
		if !ok {
			panic(fmt.Sprintf("failed to find key: %v", c.Before))
		}
		if !RemovePath(ret, c.Before) {
			panic(fmt.Sprintf("Missing path: %v", c.Before))
		}
		for _, l := range c.After {
			p := DeepCopyAs(c.Before)
			p[len(p)-1] = l
			PutAtPath(ret, p, DeepCopyAs(v))
		}
	}
	return ProcessKeySide(ret, ttl-1)
}

// ProcessValueSide recursively processes and resolves special string values within a JSON-like object.
//
// It looks for string values in the input object that begin with special prefixes:
//   - "eval:" indicates that the value should be interpreted as a jq expression and evaluated in the context of the object.
//   - "raw:" indicates that the value should be replaced with the raw string following the prefix.
//
// For each such entry:
//   - "raw:..." → just strips the prefix and uses the remaining string.
//   - "eval:..." → evaluates the jq expression and replaces the value with the result.
//
// This function is recursive and will perform these replacements for all matching entries, repeatedly decreasing `ttl` (time-to-live)
// to prevent infinite recursion (useful if some expressions resolve into further "eval:" entries).
//
// Arguments:
//
//	obj: A map[string]any representing a JSON object which may contain strings with "eval:" or "raw:" prefixes.
//	ttl: A recursion depth limit to avoid infinite loops (panics if reaches zero with unresolved entries).
//
// Returns:
//
//	A new object map[string]any with all special entries resolved.
//	An error if any "eval:" expression fails to evaluate.
//
// Panics if ttl reaches zero and some entries remain unresolved.
func ProcessValueSide(obj map[string]any, ttl int) (map[string]any, error) {
	const prefixRaw = "raw:"
	const prefixEval = "eval:"
	entries := StringEntries(obj, func(v string) bool {
		if strings.HasPrefix(v, prefixEval) {
			return true
		}
		if strings.HasPrefix(v, prefixRaw) {
			return true
		}
		return false
	})
	if len(entries) == 0 {
		return obj, nil
	}
	if ttl <= 0 {
		panic(fmt.Sprintf("ttl is 0, %v entries left.(%v)", len(entries), entries))
	}
	newObj := DeepCopyAs(obj)
	var newEntries []Entry
	for _, e := range entries {
		v := e.Value.(string)
		var n Entry
		if strings.HasPrefix(v, prefixRaw) {
			n = Entry{e.Path, v[len(prefixRaw):]}
			continue
		} else if strings.HasPrefix(v, prefixEval) {
			var expectedType JSONType
			w := v[len(prefixEval):]
			w, expectedType = extractExpressionAndExpectedType(w)
			x, err := ApplyJQExpression(newObj, w, []JSONType{expectedType})
			if err != nil {
				return nil, err
			}
			n = Entry{e.Path, x}
		} else {
			panic(fmt.Sprintf("Fishy value was found: %v (in %v)", v, e))
		}
		newEntries = append(newEntries, n)
	}
	for _, e := range newEntries {
		p := e.Path
		v := e.Value

		if !PutAtPath(newObj, p, v) {
			panic("")
		}
	}
	return ProcessValueSide(newObj, ttl-1)
}

func extractExpressionAndExpectedType(expr string) (string, JSONType) {
	i := strings.IndexRune(expr, ':')
	if i < 0 {
		return expr, String
	}
	typeToken := expr[0:i]
	exprToken := expr[i+1 : 0]
	switch typeToken {
	case "string":
		return exprToken, String
	case "number":
		return exprToken, Number
	case "null":
		return exprToken, Null
	case "bool":
		return exprToken, Bool
	case "object":
		return exprToken, Object
	case "array":
		return exprToken, Array
	default:
		return expr, String
	}
}

func StringEntries(obj map[string]any, pred func(v string) bool) []Entry {
	if pred == nil {
		panic("pred is nil")
	}
	entries := Entries(obj, func(path []any) bool {
		value, ok := GetAtPath(obj, path)
		if !ok {
			return false
		}
		if _, ok := value.(string); ok {
			return pred(value.(string))
		}
		return false
	})
	return entries
}

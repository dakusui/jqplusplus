package internal

import (
	"fmt"
	"github.com/itchyny/gojq"
)

type JSONType int

const (
	Null = iota
	Boolean
	String
	Number
	Array
	Object
)

func (t JSONType) String() string {
	switch t {
	case Null:
		return "null"
	case Boolean:
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
	expectedType JSONType,
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
	switch expectedType {
	case String:
		if val, ok := result.(string); ok {
			return val, nil
		}
	case Array:
		if val, ok := result.([]any); ok {
			return val, nil
		}
	case Object:
		if val, ok := result.(map[string]any); ok {
			return val, nil
		}
	case Number:
		switch v := result.(type) {
		case float64:
			return v, nil
		case int:
			return v, nil
		case int64:
			return v, nil // you may see this depending on platform / custom funcs
		default:
			return nil, fmt.Errorf("result type mismatch: expected %s but got %T", expectedType, result)
		}
	case Boolean:
		if val, ok := result.(bool); ok {
			return val, nil
		}
	case Null:
		if result == nil {
			return nil, nil
		}
	default:
		return nil, fmt.Errorf("unsupported expected type: %v", expectedType)
	}

	return nil, fmt.Errorf("result type mismatch: expected %v but got %T", expectedType.String(), result)
}

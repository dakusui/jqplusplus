package internal

import (
	"fmt"
	"github.com/itchyny/gojq"
)

// applyJQExpression applies a jq expression to the provided input object, validates the result type,
// and returns it in the specified type.
// applyJQExpression applies a jq expression to the provided input object, validates the result type,
// and returns it in the specified type.
//
// NOTE: Custom jq functions/modules are enabled by compiling the parsed query with compiler options
// (e.g., gojq.WithFunction, gojq.WithModuleLoader, ...).
func applyJQExpression(
	input any,
	expression string,
	expectedType string,
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
	case "string":
		if val, ok := result.(string); ok {
			return val, nil
		}
	case "array":
		if val, ok := result.([]any); ok {
			return val, nil
		}
	case "object":
		if val, ok := result.(map[string]any); ok {
			return val, nil
		}
	case "number":
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
	case "boolean":
		if val, ok := result.(bool); ok {
			return val, nil
		}
	case "null":
		if result == nil {
			return nil, nil
		}
	default:
		return nil, fmt.Errorf("unsupported expected type: %s", expectedType)
	}

	return nil, fmt.Errorf("result type mismatch: expected %s but got %T", expectedType, result)
}

package main

import (
	"fmt"
	"github.com/itchyny/gojq"
)

func main() {
	v, err := applyJQExpression(map[string]any{"a": "Hello", "b": "X"}, `.a|gsub("l"; "X")`, "string")
	if err != nil {
		panic(err)
	}
	fmt.Println(v)
	w, err := applyJQExpression(map[string]any{"a": "Hello", "b": "X"}, `.a|gsub("l"; "X")`, "string")
	if err != nil {
		panic(err)
	}
	fmt.Println(w)
}

// applyJQExpression applies a jq expression to the provided input object, validates the result type,
// and returns it in the specified type.
func applyJQExpression(input any, expression string, expectedType string) (any, error) {
	// Parse the jq expression
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jq expression: %w", err)
	}

	// Run the jq query
	iter := query.Run(input)

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
		if val, ok := result.(float64); ok {
			return val, nil
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

package internal

import (
	"reflect"
	"testing"
)

func TestEval_a(t *testing.T) {
	input := map[string]any{"a": "Hello", "b": "X", "c": 1.0, "d": 234, "e": 12345678901234567}
	expression := `.a|gsub("l"; "X")`
	expected := "HeXXo"
	v, err := ApplyJQExpression(input, expression, []JSONType{String})
	if err != nil {
		t.Errorf("Failed to apply '%s' to '%s': %s", input, expression, err)
	}
	if v != "HeXXo" {
		t.Errorf("Expected '%s' but got '%s'", expected, v)
	}
}

func TestEval_c(t *testing.T) {
	input := map[string]any{"a": "Hello", "b": "X", "c": 1.0, "d": 234, "e": 12345678901234567}
	expression := `.c`
	expected := 1.0
	v, err := ApplyJQExpression(input, expression, []JSONType{Number})
	if err != nil {
		t.Errorf("Failed to apply '%s' to '%s': %s", input, expression, err)
	}
	if v != expected {
		t.Errorf("Expected '%f' but got '%s'", expected, v)
	}
}
func TestEval_d(t *testing.T) {
	input := map[string]any{"a": "Hello", "b": "X", "c": 1.0, "d": 234, "e": 12345678901234567}
	expression := `.d`
	expected := 234
	v, err := ApplyJQExpression(input, expression, []JSONType{Number})
	if err != nil {
		t.Errorf("Failed to apply '%s' to '%s': %s", input, expression, err)
	}
	if v != expected {
		t.Errorf("Expected '%d' but got '%s'", expected, v)
	}
}
func TestEval_e(t *testing.T) {
	input := map[string]any{"a": "Hello", "b": "X", "c": 1.0, "d": 234, "e": 12345678901234567}
	expression := `.e`
	expected := 12345678901234567
	v, err := ApplyJQExpression(input, expression, []JSONType{Number})
	if err != nil {
		t.Errorf("Failed to apply '%s' to '%s': %s", input, expression, err)
	}
	if v != expected {
		t.Errorf("Expected '%d' but got '%s'", expected, v)
	}
}

func TestProcessValueSide(t *testing.T) {
	input := map[string]any{"a": "Hello", "X": "eval:.a"}
	//expected := "processedKey" // Replace this with the expected outcome of the input

	result, err := ProcessValueSide(input, 7)
	if err != nil {
		t.Errorf("ProcessKeySide failed for input '%s' with error: %s", input, err)
	}

	expected := map[string]any{"a": "Hello", "X": "Hello"}
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected '%s', but got '%s'", expected, result)
	}
}

func TestProcessValueSide_2(t *testing.T) {
	input := map[string]any{"a": "Hello", "X": "eval:$cur"}
	//expected := "processedKey" // Replace this with the expected outcome of the input

	result, err := ProcessValueSide(input, 7)
	if err != nil {
		t.Errorf("ProcessKeySide failed for input '%s' with error: %s", input, err)
	}

	expected := map[string]any{"a": "Hello", "X": "HELLO"}
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected '%s', but got '%s'", expected, result)
	}
}

func TestProcessKeySide(t *testing.T) {
	input := map[string]any{"a": "Hello", "eval:.a": "X"}
	//expected := "processedKey" // Replace this with the expected outcome of the input

	result, err := ProcessKeySide(input, 7)
	if err != nil {
		t.Errorf("ProcessKeySide failed for input '%s' with error: %s", input, err)
	}

	expected := map[string]any{"Hello": "X", "a": "Hello"}
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected '%s', but got '%s'", expected, result)
	}
}

func TestProcessKeySide_2(t *testing.T) {
	input := map[string]any{"a": []string{"Hello", "Howdy"}, "eval:.a": "X"}
	//expected := "processedKey" // Replace this with the expected outcome of the input

	result, err := ProcessKeySide(input, 7)
	if err != nil {
		t.Errorf("ProcessKeySide failed for input '%s' with error: %s", input, err)
	}

	expected := map[string]any{"Hello": "X", "Howdy": "X", "a": []string{"Hello", "Howdy"}}
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected '%s', but got '%s'", expected, result)
	}
}

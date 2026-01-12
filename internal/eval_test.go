package internal

import (
	"testing"
)

func TestEva_a(t *testing.T) {
	input := map[string]any{"a": "Hello", "b": "X", "c": 1.0, "d": 234, "e": 12345678901234567}
	expression := `.a|gsub("l"; "X")`
	expected := "HeXXo"
	v, err := ApplyJQExpression(input, expression, String)
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
	v, err := ApplyJQExpression(input, expression, Number)
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
	v, err := ApplyJQExpression(input, expression, Number)
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
	v, err := ApplyJQExpression(input, expression, Number)
	if err != nil {
		t.Errorf("Failed to apply '%s' to '%s': %s", input, expression, err)
	}
	if v != expected {
		t.Errorf("Expected '%d' but got '%s'", expected, v)
	}
}

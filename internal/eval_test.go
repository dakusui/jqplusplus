package internal

import (
	"fmt"
	"testing"
)

func TestEva_a(t *testing.T) {
	input := map[string]any{"a": "Hello", "b": "X", "c": 1.0, "d": 234, "e": 12345678901234567}
	w, err := applyJQExpression(input, `.a|gsub("l"; "X")`, "string")
	if err != nil {
		panic(err)
	}
	fmt.Println(w)
}

func TestEval_c(t *testing.T) {
	input := map[string]any{"a": "Hello", "b": "X", "c": 1.0, "d": 234, "e": 12345678901234567}
	x, err := applyJQExpression(input, `.c`, "number")
	if err != nil {
		panic(err)
	}
	fmt.Println(x)
}
func TestEval_d(t *testing.T) {
	input := map[string]any{"a": "Hello", "b": "X", "c": 1.0, "d": 234, "e": 12345678901234567}
	y, err := applyJQExpression(input, `.d`, "number")
	if err != nil {
		fmt.Printf("(%s)\n", y)
		panic(err)
	}
	fmt.Println(y)
}
func TestEval_e(t *testing.T) {
	input := map[string]any{"a": "Hello", "b": "X", "c": 1.0, "d": 234, "e": 12345678901234567}
	z, err := applyJQExpression(input, `.e`, "number")
	if err != nil {
		fmt.Printf("(%s)\n", z)
		panic(err)
	}
	fmt.Println(z)
}

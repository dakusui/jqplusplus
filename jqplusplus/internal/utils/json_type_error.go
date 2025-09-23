package utils

import "fmt"

type JsonTypeError struct {
	jsonErr
	expected string
	actual   string
}

func NewJsonTypeError(path string, expected string, actual string) *JsonTypeError {
	return &JsonTypeError{jsonErr{path}, expected, actual}
}

func (e *JsonTypeError) Error() string {
	return fmt.Sprintf("Expected '%s' but got '%s'", e.expected, e.actual)
}

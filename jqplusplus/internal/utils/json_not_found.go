package utils

import "fmt"

type JsonNotFound struct {
	jsonErr
}

func NewJsonNotFound(path string) *JsonNotFound {
	return &JsonNotFound{jsonErr{path}}
}

func (e *JsonNotFound) Error() string {
	return fmt.Sprintf("Not found at '%s'", e.path)
}

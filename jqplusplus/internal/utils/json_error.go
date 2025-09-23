package utils

type JsonError interface {
	error
	JsonPath()
}

type jsonErr struct {
	path string
}

func (e *jsonErr) JSONPath() string {
	return e.path
}

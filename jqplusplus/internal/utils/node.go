package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type ObjectNode struct {
	objectNode map[string]interface{}
}

func (n *ObjectNode) Has(key string) bool {
	_, ok := n.objectNode[key]
	if ok {
		return true
	}
	return false
}

func (n *ObjectNode) Keys() []string {
	keys := make([]string, 0, len(n.objectNode))
	for k := range n.objectNode {
		keys = append(keys, k)
	}
	return keys
}

func (n *ObjectNode) Get(key string) interface{} {
	v, ok := n.objectNode[key]
	if !ok {
		panic("Unknown key was given: '" + key + "'")
	}
	return v
}

func (n *ObjectNode) Find(pexp string) (interface{}, error) {
	path, err := PexpToPath(pexp)
	if err != nil {
		return nil, err
	}
	var ret any
	ret = n
	for _, k := range path {
		switch t := k.(type) {
		case string:
			if !n.Has(k.(string)) {
				return nil, fmt.Errorf("path %q is not an object", pexp)
			}
			ret = n.Get(k.(string))
			break
		case int:
			break
		default:
			msg, _ := fmt.Printf("unknown type: %T", t)
			panic(msg)
		}
	}
	return ret, nil
}

func PathToPexp(segments []any) string {
	var b bytes.Buffer
	for _, s := range segments {
		switch v := s.(type) {
		case string:
			// ."key" with proper escaping; jq prints quotes inside
			b.WriteByte('.')
			b.WriteString(strconv.Quote(v))
		case int:
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(v))
			b.WriteByte(']')
		case json.Number:
			// array indices are ints in our traversal; this is here just in case
			b.WriteByte('[')
			b.WriteString(v.String())
			b.WriteByte(']')
		default:
			// should not occur; keep behavior defined
			b.WriteByte('.')
			b.WriteString(strconv.Quote(fmtAny(v)))
		}
	}
	return b.String()
}

func fmtAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// PexpToPath parses a jq-style pexp (."foo"[0]."bar")
// into a slice of path segments ([]any).
// Strings become Go strings, [N] become ints.
func PexpToPath(pexp string) ([]any, error) {
	var path []any
	i := 0
	for i < len(pexp) {
		switch pexp[i] {
		case '.':
			// expect ."<key>"
			i++
			if i >= len(pexp) || pexp[i] != '"' {
				return nil, fmt.Errorf("expected '\"' after '.' at %d", i)
			}
			i++
			var sb strings.Builder
			for {
				if i >= len(pexp) {
					return nil, fmt.Errorf("unterminated string key")
				}
				ch := pexp[i]
				if ch == '"' {
					break
				}
				if ch == '\\' {
					if i+1 >= len(pexp) {
						return nil, fmt.Errorf("bad escape at %d", i)
					}
					sb.WriteByte(pexp[i+1])
					i += 2
					continue
				}
				sb.WriteByte(ch)
				i++
			}
			path = append(path, sb.String())
			i++ // skip closing quote

		case '[':
			// parse number until ']'
			i++
			start := i
			for i < len(pexp) && unicode.IsDigit(rune(pexp[i])) {
				i++
			}
			if start == i {
				return nil, fmt.Errorf("empty index at %d", start)
			}
			numStr := pexp[start:i]
			if i >= len(pexp) || pexp[i] != ']' {
				return nil, fmt.Errorf("expected ']' after index at %d", i)
			}
			idx, err := strconv.Atoi(numStr)
			if err != nil {
				return nil, fmt.Errorf("bad index %q: %v", numStr, err)
			}
			path = append(path, idx)
			i++ // skip ']'

		default:
			return nil, fmt.Errorf("unexpected char %q at %d", pexp[i], i)
		}
	}
	return path, nil
}

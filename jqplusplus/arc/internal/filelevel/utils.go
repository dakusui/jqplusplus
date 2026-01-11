package filelevel

import (
	"fmt"
	"strconv"
	"strings"
)

func Reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func JSONPaths(obj map[string]any, pred func([]any) bool) [][]any {
	var paths [][]any
	walkAnyPath(nil, obj, &paths)
	return Filter(paths, pred)
}

func walkAnyPath(prefix []any, v any, out *[][]any) {
	switch x := v.(type) {

	case map[string]any:
		for k, v2 := range x {
			p := append(prefix, k) // k is string
			*out = append(*out, p)
			walkAnyPath(p, v2, out)
		}

	case []any:
		for i, v2 := range x {
			p := append(prefix, i) // i is int
			*out = append(*out, p)
			walkAnyPath(p, v2, out)
		}
	}
}

// GetAtPath returns the value at `path` inside `root`.
// Path segments: string => map key, int => array index.
// ok=false if the path can't be followed.
func GetAtPath(root any, path []any) (val any, ok bool) {
	cur := root

	for _, seg := range path {
		switch s := seg.(type) {

		case string:
			m, isMap := cur.(map[string]any)
			if !isMap {
				// also support map[string]interface{} if your code uses that
				m2, isMap2 := cur.(map[string]interface{})
				if !isMap2 {
					return nil, false
				}
				next, exists := m2[s]
				if !exists {
					return nil, false
				}
				cur = next
				continue
			}
			next, exists := m[s]
			if !exists {
				return nil, false
			}
			cur = next

		case int:
			arr, isArr := cur.([]any)
			if !isArr {
				arr2, isArr2 := cur.([]interface{})
				if !isArr2 {
					return nil, false
				}
				if s < 0 || s >= len(arr2) {
					return nil, false
				}
				cur = arr2[s]
				continue
			}
			if s < 0 || s >= len(arr) {
				return nil, false
			}
			cur = arr[s]

		default:
			panic(fmt.Sprintf("unsupported path segment type: %T", seg))
		}
	}

	return cur, true
}

// SetAtPath sets `value` at `path` inside `root`.
// It mutates `root` only if the whole path is valid.
// Path segments: string => map key, int => array index.
func SetAtPath(root any, path []any, value any) (ok bool) {
	if len(path) == 0 {
		return false
	}

	cur := root

	// Walk to the parent of the target.
	for i := 0; i < len(path)-1; i++ {
		seg := path[i]

		switch s := seg.(type) {
		case string:
			m, ok := cur.(map[string]any)
			if !ok {
				// also tolerate map[string]interface{} if you still have it
				m2, ok2 := cur.(map[string]interface{})
				if !ok2 {
					return false
				}
				next, exists := m2[s]
				if !exists {
					return false
				}
				cur = next
				continue
			}
			next, exists := m[s]
			if !exists {
				return false
			}
			cur = next

		case int:
			if s < 0 {
				return false
			}
			arr, ok := cur.([]any)
			if !ok {
				arr2, ok2 := cur.([]interface{})
				if !ok2 {
					return false
				}
				if s >= len(arr2) {
					return false
				}
				cur = arr2[s]
				continue
			}
			if s >= len(arr) {
				return false
			}
			cur = arr[s]

		default:
			panic(fmt.Sprintf("unsupported path segment type: %T", seg))
		}
	}

	// Set on the last segment.
	last := path[len(path)-1]
	switch s := last.(type) {
	case string:
		m, ok := cur.(map[string]any)
		if !ok {
			m2, ok2 := cur.(map[string]interface{})
			if !ok2 {
				return false
			}
			if _, exists := m2[s]; !exists {
				return false // strict: key must already exist
			}
			m2[s] = value
			return true
		}
		if _, exists := m[s]; !exists {
			return false // strict: key must already exist
		}
		m[s] = value
		return true

	case int:
		if s < 0 {
			return false
		}
		arr, ok := cur.([]any)
		if !ok {
			arr2, ok2 := cur.([]interface{})
			if !ok2 {
				return false
			}
			if s >= len(arr2) {
				return false
			}
			arr2[s] = value
			return true
		}
		if s >= len(arr) {
			return false
		}
		arr[s] = value
		return true

	default:
		panic(fmt.Sprintf("unsupported path segment type: %T", last))
	}
}

func newContainer(next any) any {
	switch next.(type) {
	case string:
		return map[string]any{}
	case int:
		return []any{}
	default:
		panic(fmt.Sprintf("unsupported path segment type: %T", next))
	}
}

func Filter[T any](in []T, pred func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

func Map[T any, R any](in []T, f func(T) R) []R {
	out := make([]R, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

func DistinctBy[T any, K comparable](in []T, key func(T) K) []T {
	seen := make(map[K]struct{}, len(in))
	out := make([]T, 0, len(in))

	for _, v := range in {
		k := key(v)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, v)
	}
	return out
}

func DropLast[T any](in []T) []T {
	if len(in) == 0 {
		return nil
	}
	out := make([]T, len(in)-1)
	copy(out, in[:len(in)-1])
	return out
}

func lastElementIsOneOf(v ...string) func(p []any) bool {
	return func(p []any) bool {
		if len(p) == 0 {
			return false
		}
		s, ok := p[len(p)-1].(string)
		if !ok {
			return false
		}
		for _, v := range v {
			if s == v {
				return true
			}
		}
		return false
	}
}

func pathKey(p []any) string {
	var b strings.Builder
	for _, v := range p {
		switch x := v.(type) {
		case string:
			b.WriteString("s:")
			b.WriteString(x)
		case int:
			b.WriteString("i:")
			b.WriteString(strconv.Itoa(x))
		default:
			panic("unsupported type in path")
		}
		b.WriteByte('|')
	}
	return b.String()
}

// ToAnySlice converts a []T into a []any by copying elements.
func ToAnySlice[T any](xs []T) []any {
	out := make([]any, len(xs))
	for i, v := range xs {
		out[i] = v
	}
	return out
}

func DeepCopyMap(originalMap map[string]interface{}) map[string]interface{} {
	copyMap := make(map[string]interface{})
	for key, value := range originalMap {
		switch v := value.(type) {
		case map[string]interface{}:
			// If the value is a nested map, call DeepCopyMap recursively
			copyMap[key] = DeepCopyMap(v)
		default:
			// If the value is not a map, just assign it to the new map
			copyMap[key] = value
		}
	}
	return copyMap
}

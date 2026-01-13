package internal

import "fmt"

type Entry struct {
	Path  []any
	Value any
}

func Entries(obj map[string]any, pred func([]any) bool) []Entry {
	paths := Paths(obj, pred)
	return Map(paths, func(path []any) Entry {
		if val, ok := GetAtPath(obj, path); ok {
			return Entry{Path: path, Value: val}
		}
		return Entry{}
	})
}

// Paths returns all JSON paths in `obj` that satisfy `pred`.
func Paths(obj map[string]any, pred func([]any) bool) [][]any {
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

// DeepCopy creates a deep copy of the given value.
// It recursively copies maps and slices, while preserving primitive values.
func DeepCopy(v any) any {
	switch x := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(x))
		for k, val := range x {
			result[k] = DeepCopy(val)
		}
		return result
	case []any:
		result := make([]any, len(x))
		for i, val := range x {
			result[i] = DeepCopy(val)
		}
		return result
	default:
		// Primitive types (string, int, float64, bool, nil) are returned as-is
		return v
	}
}

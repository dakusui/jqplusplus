package internal

import (
	"fmt"
	"sort"
)

type Entry struct {
	Path  []any
	Value any
}

func Entries(obj map[string]any, pred func([]any) bool) []Entry {
	paths := Paths(obj, pred)
	if len(paths) == 0 {
		fmt.Println("paths is empty!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	} else {
		fmt.Println("paths is NOT empty!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	}
	return Map(paths, func(path []any) Entry {
		if val, ok := GetAtPath(obj, path); ok {
			return Entry{Path: path, Value: val}
		}
		return Entry{}
	})
}

// Paths returns all JSON paths in `Obj` that satisfy `pred`.
func Paths(obj map[string]any, pred func([]any) bool) [][]any {
	var paths [][]any
	walkAnyPath(nil, obj, &paths)
	if len(paths) == 0 {
		fmt.Printf("Paths: paths is empty\n")
	} else {
		fmt.Printf("Paths: paths is NOT empty\n")
	}
	savedPaths := DeepCopyAs(paths)
	// Sort paths in dictionary order
	sort.SliceStable(savedPaths, func(i, j int) bool {
		x, _ := PathArrayToPathExpression(savedPaths[i])
		y, _ := PathArrayToPathExpression(savedPaths[j])
		return x < y
	})
	return Filter(savedPaths, pred)
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
func GetAtPath(root any, path []any) (any, bool) {
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

// PutAtPath sets `value` at `path` inside `root`, creating intermediate maps or arrays as necessary.
// Path segments: string => map key, int => array index.
// Returns true if the operation was successful or false if any intermediate types are incompatible.
func PutAtPath(root any, path []any, value any) bool {
	if len(path) == 0 {
		return false
	}

	cur := root

	// Traverse the path.
	for i := 0; i < len(path)-1; i++ {
		seg := path[i]

		switch s := seg.(type) {
		case string:
			// Handle map[string]any.
			m, ok := cur.(map[string]any)
			if !ok {
				m2, ok2 := cur.(map[string]interface{})
				if ok2 {
					m = make(map[string]any, len(m2))
					for k, v := range m2 {
						m[k] = v
					}
					cur = m
				} else {
					return false
				}
			}

			// Create missing map if necessary.
			if _, exists := m[s]; !exists {
				m[s] = make(map[string]any)
			}
			cur = m[s]

		case int:
			// Handle []any.
			if s < 0 {
				return false
			}
			arr, ok := cur.([]any)
			if !ok {
				arr2, ok2 := cur.([]interface{})
				if ok2 {
					arr = make([]any, len(arr2))
					for i, v := range arr2 {
						arr[i] = v
					}
					cur = arr
				} else {
					return false
				}
			}

			// Expand the array if the index is out of bounds.
			if s >= len(arr) {
				for j := len(arr); j <= s; j++ {
					arr = append(arr, nil)
				}
			}
			cur = arr[s]
			if arr[s] == nil {
				arr[s] = make(map[string]any)
			}

		default:
			panic(fmt.Sprintf("unsupported path segment type: %T", seg))
		}
	}

	// Set the value at the last segment.
	last := path[len(path)-1]
	switch s := last.(type) {
	case string:
		// Handle map[string]any.
		m, ok := cur.(map[string]any)
		if !ok {
			m2, ok2 := cur.(map[string]interface{})
			if ok2 {
				m = make(map[string]any, len(m2))
				for k, v := range m2 {
					m[k] = v
				}
				cur = m
			} else {
				return false
			}
		}
		m[s] = value
		return true

	case int:
		// Handle []any.
		if s < 0 {
			return false
		}
		arr, ok := cur.([]any)
		if !ok {
			arr2, ok2 := cur.([]interface{})
			if ok2 {
				arr = make([]any, len(arr2))
				for i, v := range arr2 {
					arr[i] = v
				}
				cur = arr
			} else {
				return false
			}
		}

		// Expand the array if the index is out of bounds.
		if s >= len(arr) {
			for j := len(arr); j <= s; j++ {
				arr = append(arr, nil)
			}
		}
		arr[s] = value
		return true

	default:
		panic(fmt.Sprintf("unsupported path segment type: %T", last))
	}
}

// RemovePath removes the entry at the specified path from the given object or array.
// Returns true if removal succeeded, false if the path could not be resolved.
func RemovePath(root any, path []any) bool {
	if len(path) == 0 {
		// Cannot remove root itself
		return false
	}
	cur := root
	// Descend to the parent of the item to remove
	for i := 0; i < len(path)-1; i++ {
		seg := path[i]
		switch s := seg.(type) {
		case string:
			m, ok := cur.(map[string]any)
			if !ok {
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
			arr, ok := cur.([]any)
			if !ok {
				arr2, ok2 := cur.([]interface{})
				if !ok2 {
					return false
				}
				if s < 0 || s >= len(arr2) {
					return false
				}
				cur = arr2[s]
				continue
			}
			if s < 0 || s >= len(arr) {
				return false
			}
			cur = arr[s]
		default:
			return false
		}
	}
	// Now remove the final segment
	last := path[len(path)-1]
	switch s := last.(type) {
	case string:
		if m, ok := cur.(map[string]any); ok {
			if _, exists := m[s]; !exists {
				return false
			}
			delete(m, s)
			return true
		}
		if m2, ok2 := cur.(map[string]interface{}); ok2 {
			if _, exists := m2[s]; !exists {
				return false
			}
			delete(m2, s)
			return true
		}
		return false
	case int:
		if s < 0 {
			return false
		}
		if arr, ok := cur.([]any); ok {
			if s >= len(arr) {
				return false
			}
			// Remove the entry by slicing
			arr = append(arr[:s], arr[s+1:]...)
			// Now assign the sliced result back into the parent structure
			// - This works only if parent's reference is available (backwards assignment)
			// - But in Go, slices are not automatically updated in the parent.
			// So we need to mutate the parent directly
			// (since the parent pointer is lost here, we can't update; thus we can only nil the element)
			// As a fallback, set the entry to nil
			cur.([]any)[s] = nil
			return true
		}
		if arr2, ok2 := cur.([]interface{}); ok2 {
			if s >= len(arr2) {
				return false
			}
			arr2[s] = nil
			return true
		}
		return false
	default:
		return false
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

func DeepCopyAs[T any](v T) T {
	return DeepCopy(any(v)).(T)
}

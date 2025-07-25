package inheritance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadAndResolve loads a JSON file, resolves inheritance, and returns the merged result as a map.
func LoadAndResolve(filename string) (map[string]interface{}, error) {
	visited := map[string]bool{}
	return loadAndResolveRecursive(filename, visited)
}

// loadAndResolveRecursive loads a JSON file, resolves $extends recursively, and merges parents.
func loadAndResolveRecursive(filename string, visited map[string]bool) (map[string]interface{}, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	if visited[absPath] {
		return nil, fmt.Errorf("circular inheritance detected: %s", absPath)
	}
	visited[absPath] = true

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	// Check for $extends
	extends, ok := obj["$extends"]
	if ok {
		parentFiles, err := parseExtendsField(extends)
		if err != nil {
			return nil, err
		}
		for _, parent := range parentFiles {
			parentObj, err := loadAndResolveRecursive(filepath.Join(filepath.Dir(absPath), parent), visited)
			if err != nil {
				return nil, err
			}
			obj = mergeObjects(parentObj, obj)
		}
		delete(obj, "$extends")
	}

	return obj, nil
}

// parseExtendsField parses the $extends field, which can be a string or array of strings.
func parseExtendsField(val interface{}) ([]string, error) {
	switch v := val.(type) {
	case string:
		return []string{v}, nil
	case []interface{}:
		var result []string
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("$extends array must contain only strings")
			}
			result = append(result, str)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("$extends must be a string or array of strings")
	}
}

// mergeObjects merges parent and child objects, with child values taking precedence.
func mergeObjects(parent, child map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range parent {
		result[k] = v
	}
	for k, v := range child {
		if k == "$extends" {
			continue
		}
		// If both are maps, merge recursively
		if pv, ok := result[k].(map[string]interface{}); ok {
			if cv, ok := v.(map[string]interface{}); ok {
				result[k] = mergeObjects(pv, cv)
				continue
			}
		}
		result[k] = v
	}
	return result
}

// ListSearchPaths generates the list of directories to search when resolving a file.
func ListSearchPaths(currentFile string) []string {
	// TODO: implement
	return nil
}

// ResolveFilePath finds the full path of a referenced file from a list of directories.
func ResolveFilePath(name string, searchPaths []string) (string, error) {
	// TODO: implement
	return "", nil
}

// LoadFileAsRawJSON loads and parses a file (JSON, YAML, etc.) into a gojq-compatible object.
func LoadFileAsRawJSON(path string, formatOpt string) (map[string]interface{}, error) {
	// TODO: implement
	return nil, nil
}

// ExtractExtendsField returns the list of $extends references from a JSON object.
func ExtractExtendsField(obj map[string]interface{}) ([]string, error) {
	// TODO: implement
	return nil, nil
}

// ExtractIncludesField returns the list of $includes references from a JSON object.
func ExtractIncludesField(obj map[string]interface{}) ([]string, error) {
	// TODO: implement
	return nil, nil
}

// MergePolicy defines the policy for merging objects.
type MergePolicy int

const (
	MergePolicyDefault MergePolicy = iota
	// Add more policies as needed
)

// MergeObjects merges two JSON objects (b overrides a by default).
func MergeObjects(a, b map[string]interface{}, policy MergePolicy) map[string]interface{} {
	// TODO: implement
	return nil
}

// MergeMultiple merges a list of JSON objects in order (or reverse).
func MergeMultiple(objs []map[string]interface{}, reverse bool, policy MergePolicy) map[string]interface{} {
	// TODO: implement
	return nil
}

// ResolveInheritance is the top-level interface: load file and resolve all $extends and $includes.
func ResolveInheritance(path string) (map[string]interface{}, error) {
	// TODO: implement
	return nil, nil
}

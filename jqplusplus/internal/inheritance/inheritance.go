package inheritance

import (
	"encoding/json"
	"fmt"
	"github.com/dakusui/jqplusplus/jqplusplus/internal/app"
	"github.com/dakusui/jqplusplus/jqplusplus/internal/utils"
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
		parentFiles, err := parseExtendsField(extends, false)
		if err != nil {
			return nil, err
		}
		var mergedParents map[string]interface{}
		for i, parent := range parentFiles {
			parentObj, err := loadAndResolveRecursive(filepath.Join(filepath.Dir(absPath), parent), visited)
			if err != nil {
				return nil, err
			}
			if i == 0 {
				mergedParents = parentObj
			} else {
				mergedParents = mergeObjects(mergedParents, parentObj)
			}
		}
		if mergedParents != nil {
			obj = mergeObjects(mergedParents, obj)
		}
		delete(obj, "$extends")
	}

	return obj, nil
}

// parseExtendsField parses the $extends field, which can be a string or array of strings.
func parseExtendsField(val interface{}, reverseOrder bool) ([]string, error) {
	switch v := val.(type) {
	case []interface{}:
		var result []string
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("$extends array must contain only strings")
			}
			if reverseOrder {
				result = append(result, str)
			} else {
				result = utils.Insert(result, 0, str)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("$extends must be an array of strings")
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

type MergeType int

const (
	MergeIncludes MergeType = iota
	MergeExtends
)

// ListSearchPaths generates the list of directories to search when resolving a file.
// This function works in the following way:
// 1. Resolve the directory of the current file
// 2. Add the directory of the current file to the search paths
// 3. Reads environment variable JF_PATH
// 4. Splits on :
// 5. Returns the list of directories
// 6. Prepend the search paths with the directories in JF_PATH
func ListSearchPaths(app app.App, currentFile string) []string {
	return append(app.SearchPaths(), filepath.Dir(currentFile))
}

// ResolveFilePath finds the full path of a referenced file from a list of directories.
// This function works in the following way:
// 1. Iterate over the search paths
//  1. Check if the path exists.
//  2. If it exists, check if it is a true file.
//  3. If it is a true file, return it.
//  4. If it is a directory, return an error.
//
// 2. If the file is not found, return an error.
func ResolveFilePath(name string, searchPaths []string) (string, error) {
	// Iterate over the search paths
	for _, path := range searchPaths {
		// Check if the path exists.
		// 	If exists, return it.
		fullPath := filepath.Join(path, name)
		if _, err := os.Stat(fullPath); err == nil {
			// If it is a true file, return it.
			if !os.IsNotExist(err) {
				return fullPath, nil
			}
			// If it is a directory, return an error.
			return "", fmt.Errorf("file is a directory: %s", fullPath)
		}
	}
	return "", fmt.Errorf("file not found: %s", name)
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
	result := make(map[string]interface{})
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		if av, ok := result[k].(map[string]interface{}); ok {
			if bv, ok := v.(map[string]interface{}); ok {
				result[k] = MergeObjects(av, bv, policy)
				continue
			}
		}
		result[k] = v
	}
	return result
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

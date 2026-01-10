package filelevel

import (
	"encoding/json"
	"fmt"
	"github.com/dakusui/jqplusplus/jqplusplus/internal/utils"
	"os"
	"path/filepath"
	"strings"
)

// LoadAndResolveInheritances loads a JSON file, resolves filelevel, and returns the merged result as a map.
func LoadAndResolveInheritances(filename string, searchPaths []string) (map[string]interface{}, error) {
	visited := map[string]bool{}
	absPath, baseDir, err := ResolveFilePath(filename, "", searchPaths)
	if err != nil {
		return nil, err
	}

	visited[absPath] = true

	obj, err := loadJsonObject(absPath)
	if err != nil {
		return nil, err
	}

	tmp, err := resolveInheritances(obj, baseDir, Extends, visited, searchPaths)
	if err != nil {
		return nil, err
	}
	ret, err := resolveInheritances(tmp, baseDir, Includes, visited, searchPaths)
	return ret, err
}

// loadAndResolveRecursive loads a JSON file, resolves $extends or $includes recursively, and merges parents.
func loadAndResolveRecursive(baseDir string, targetFile string, visited map[string]bool, mergeType InheritType, searchPaths []string) (map[string]interface{}, error) {
	targetFileAbsPath, bDir, err := ResolveFilePath(targetFile, baseDir, searchPaths)
	if err != nil {
		return nil, err
	}
	if visited[targetFileAbsPath] {
		return nil, fmt.Errorf("circular filelevel detected: %s", targetFileAbsPath)
	}
	visited[targetFileAbsPath] = true

	obj, err := loadJsonObject(targetFileAbsPath)
	if err != nil {
		return nil, err
	}

	return resolveInheritances(obj, bDir, mergeType, visited, searchPaths)
}

func loadJsonObject(targetFileAbsPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(targetFileAbsPath)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func resolveInheritances(obj map[string]interface{}, baseDir string, mergeType InheritType, visited map[string]bool, searchPaths []string) (map[string]interface{}, error) {
	// Check for $extends or $includes
	inherits, ok := obj[mergeType.String()]
	if ok {
		parentFiles, err := parseInheritsField(inherits, mergeType)
		if err != nil {
			return nil, err
		}
		if mergeType.IsOrderReversed() {
			reverse(parentFiles)
		}
		var mergedParents map[string]interface{}
		for i, parent := range parentFiles {
			parentObj, err := loadAndResolveRecursive(baseDir, parent, visited, mergeType, searchPaths)
			if err != nil {
				return nil, err
			}
			if i == 0 {
				mergedParents = parentObj
			} else {
				mergedParents = mergeObjects(mergedParents, parentObj)
			}
		}
		if !mergeType.IsOrderReversed() {
			obj = mergeObjects(mergedParents, obj)
		} else {
			obj = mergeObjects(obj, mergedParents)
		}
		delete(obj, mergeType.String())
	}

	return obj, nil
}

func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// parseInheritsField parses the $extends field, which can be a string or array of strings.
func parseInheritsField(val interface{}, inherits InheritType) ([]string, error) {
	switch v := val.(type) {
	case []interface{}:
		var result []string
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s array must contain only strings: %v", inherits.String(), v)
			}
			result = utils.Insert(result, 0, str)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%s must be an array of strings", inherits.String())
	}
}

func toAbsPath(path, baseDir string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
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

type InheritType int

const (
	Includes InheritType = iota
	Extends
)

func (m InheritType) String() string {
	switch m {
	case Includes:
		return "$includes"
	case Extends:
		return "$extends"
	default:
		panic("unknown merge type")
	}
}

func (m InheritType) IsOrderReversed() bool {
	switch m {
	case Includes:
		return true
	case Extends:
		return false
	default:
		panic(fmt.Sprintf("unknown merge type: %s", m))
	}
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
func ResolveFilePath(filename string, baseDir string, searchPaths []string) (string, string, error) {
	if filepath.IsAbs(filename) {
		return filename, filepath.Dir(filename), nil
	}
	beginning := 0
	if baseDir != "" {
		beginning = -1
	}
	// Iterate over the search paths
	for i := beginning; i < len(searchPaths); i++ {
		var path string
		if i == -1 {
			path = baseDir
		} else {
			path = searchPaths[i]
		}

		// Check if the path exists.
		// 	If exists, return it.
		fullPath := filepath.Join(path, filename)
		if _, err := os.Stat(fullPath); err == nil {
			// If it is a true file, return it.
			if !os.IsNotExist(err) {
				return fullPath, filepath.Dir(fullPath), nil
			}
			// If it is a directory, return an error.
			return "", "", fmt.Errorf("file is a directory: %s", fullPath)
		}
	}
	return "", "", fmt.Errorf("file not found: %s", filename)
}

func SearchPaths() []string {
	v := os.Getenv("JF_PATH")
	return strings.Split(v, ":")
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

// MergeMultiple merges a list of JSON objects in order (or reverse).
func MergeMultiple(objs []map[string]interface{}, reverse bool, policy utils.MergePolicy) map[string]interface{} {
	// TODO: implement
	return nil
}

// ResolveInheritance is the top-level interface: load file and resolve all $extends and $includes.
func ResolveInheritance(path string) (map[string]interface{}, error) {
	// TODO: implement
	return nil, nil
}

package filelevel

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/dakusui/jqplusplus/jqplusplus/internal/utils"
	"github.com/gurkankaymak/hocon"
	"github.com/titanous/json5"
	"gopkg.in/yaml.v3"
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

	obj, err := LoadFileAsRawJSON(absPath)
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

	obj, err := LoadFileAsRawJSON(targetFileAbsPath)
	if err != nil {
		return nil, err
	}

	return resolveInheritances(obj, bDir, mergeType, visited, searchPaths)
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

type FileType string

const (
	JSON  FileType = "json"
	YAML  FileType = "yaml"
	TOML  FileType = "toml"
	JSON5 FileType = "json5"
	HCL   FileType = "hcl"
	HOCON FileType = "hocon"
)

func detectFileType(name string) (FileType, bool) {
	ext := strings.ToLower(filepath.Ext(name))

	switch ext {
	case ".json":
		return JSON, true
	case ".yaml", ".yml":
		return YAML, true
	case ".toml":
		return TOML, true
	case ".json5":
		return JSON5, true
	case ".hcl":
		return HCL, true
	case ".conf", ".hocon":
		return HOCON, true
	default:
		return "", false
	}
}

// LoadFileAsRawJSON loads and parses a file (JSON, YAML, etc.) into a gojq-compatible object.
func LoadFileAsRawJSON(path string) (map[string]interface{}, error) {
	ft, ok := detectFileType(path)
	if !ok {
		return nil, fmt.Errorf("unsupported file type: %q", filepath.Ext(path))
	}

	switch ft {
	case JSON:
		return readJSON(path)
	case YAML:
		return readYAML(path)
	case TOML:
		return readTOML(path)
	case JSON5:
		return readJSON5(path)
	case HCL:
		return nil, fmt.Errorf("unsupported file type: %q (%s)", ft, path)
	case HOCON:
		return readHOCON(path)
	default:
		return nil, fmt.Errorf("unsupported file type: %q (%s)", ft, path)
	}
}

func readJSON(targetFileAbsPath string) (map[string]interface{}, error) {
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

func readYAML(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func readTOML(path string) (map[string]interface{}, error) {
	var m map[string]interface{}
	if _, err := toml.DecodeFile(path, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func readJSON5(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json5.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// readHOCON reads a HOCON file and returns it as a JSON-compatible map.
// Top-level must be an object.
func readHOCON(path string) (map[string]interface{}, error) {
	conf, err := hocon.ParseResource(path)
	if err != nil {
		return nil, err
	}

	root := conf.GetRoot() // Value
	obj, ok := root.(hocon.Object)
	if !ok {
		return nil, fmt.Errorf("HOCON top-level must be an object")
	}

	return objectToMap(obj), nil
}

func objectToMap(o hocon.Object) map[string]interface{} {
	out := make(map[string]interface{}, len(o))
	for k, v := range o {
		out[k] = valueToAny(v)
	}
	return out
}

func arrayToSlice(a hocon.Array) []interface{} {
	out := make([]interface{}, 0, len(a))
	for _, v := range a {
		out = append(out, valueToAny(v))
	}
	return out
}

func valueToAny(v hocon.Value) interface{} {
	switch x := v.(type) {
	case hocon.Object:
		return objectToMap(x)
	case hocon.Array:
		return arrayToSlice(x)
	case hocon.String:
		return string(x)
	case hocon.Int:
		return int(x)
	case hocon.Float64:
		return float64(x)
	case hocon.Float32:
		return float32(x)
	case hocon.Boolean:
		return bool(x)
	case hocon.Null:
		return nil
	default:
		// Fallback: keep it JSON-safe as a string (covers substitutions/concatenations, etc.)
		return v.String()
	}
}

package filelevel

import (
	"encoding/json"
	"errors"
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
func LoadAndResolveInheritances(baseDir string, filename string, searchPaths []string) (map[string]any, error) {
	sessionDirectory := CreateSessionDirectory()
	defer os.RemoveAll(CreateSessionDirectory())

	return NewStdNodePoolWithBaseSearchPaths(baseDir, sessionDirectory, searchPaths).ReadNodeEntry(baseDir, filename)
}

// LoadAndResolveInheritancesRecursively loads a JSON file, resolves $extends or $includes recursively, and merges parents.
func LoadAndResolveInheritancesRecursively(baseDir string, targetFile string, nodepool NodePool) (map[string]any, error) {
	absPath, bDir, err := ResolveFilePath(targetFile, baseDir, nodepool.SearchPaths())
	if err != nil {
		return nil, err
	}
	if nodepool.IsVisited(absPath) {
		return nil, fmt.Errorf("circular filelevel detected: %s", absPath)
	}
	nodepool.MarkVisited(absPath)

	obj, err := LoadFileAsRawJSON(absPath)
	if err != nil {
		return nil, err
	}

	obj, err = resolveBothInheritances(bDir, obj, nodepool)
	if err != nil {
		return nil, err
	}

	p, err := MaterializeLocalNodes(obj, nodepool.SessionDirectory())
	delete(obj, "$local")
	nodepool.Enter(p)

	for _, p := range DistinctBy(Map(JSONPaths(obj, lastElementIsOneOf("$extends", "$includes")), DropLast[any]), pathKey) {
		internal, ok := GetAtPath(obj, ToAnySlice(p))
		if !ok {
			continue
		}
		internalObj, ok := internal.(map[string]any)
		if !ok {
			continue
		}
		internalObj, err := resolveBothInheritances(bDir, internalObj, nodepool)
		if err != nil {
			return nil, err
		}
		SetAtPath(obj, ToAnySlice(p), internalObj)
	}

	nodepool.Leave(p)
	return obj, nil
}

func resolveBothInheritances(baseDir string, obj map[string]any, nodepool NodePool) (map[string]any, error) {
	ret := obj
	var err error
	for t := range []InheritType{Extends, Includes} {
		ret, err = resolveInheritances(ret, baseDir, InheritType(t), nodepool)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func resolveInheritances(obj map[string]any, baseDir string, mergeType InheritType, nodepool NodePool) (map[string]any, error) {
	// Check for $extends or $includes
	inherits, ok := obj[mergeType.String()]
	if ok {
		parentFiles, err := parseInheritsField(inherits, mergeType)
		if err != nil {
			return nil, err
		}
		if mergeType.IsOrderReversed() {
			Reverse(parentFiles)
		}
		var mergedParents map[string]any
		for i, parent := range parentFiles {
			parentObj, err := nodepool.ReadNodeEntry(baseDir, parent)
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

// parseInheritsField parses the $extends field, which can be a string or array of strings.
func parseInheritsField(val any, inherits InheritType) ([]string, error) {
	switch v := val.(type) {
	case []any:
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
func mergeObjects(parent, child map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range parent {
		result[k] = v
	}
	for k, v := range child {
		if k == "$extends" || k == "$excludes" || k == "$local" {
			continue
		}
		// If both are maps, merge recursively
		if pv, ok := result[k].(map[string]any); ok {
			if cv, ok := v.(map[string]any); ok {
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

// MaterializeLocalNodes materializes obj["$local"] into files under dir.
// Returns dir (absolute) on success.
func MaterializeLocalNodes(obj map[string]any, localNodeDirectoryBase string) (string, error) {
	if obj == nil {
		return "", errors.New("obj is nil")
	}
	if strings.TrimSpace(localNodeDirectoryBase) == "" {
		return "", errors.New("dir is empty")
	}

	localAny, ok := obj["$local"]
	if !ok || localAny == nil {
		return "", nil
	}

	localObj, ok := localAny.(map[string]any)
	if !ok {
		return "", fmt.Errorf(`"$local" must be an object (map[string]any), got %T`, localAny)
	}

	absDir, err := os.MkdirTemp(localNodeDirectoryBase, "localnodes-")
	if err != nil {
		return "", fmt.Errorf("mkdir temp dir: %w", err)
	}

	for name, v := range localObj {
		rel, err := sanitizeRelativePath(name)
		if err != nil {
			return "", fmt.Errorf("invalid $local key %q: %w", name, err)
		}

		target := filepath.Join(absDir, rel)

		// Final guard: ensure the resulting path stays within absDir
		relToBase, err := filepath.Rel(absDir, target)
		if err != nil {
			return "", fmt.Errorf("rel check for %q: %w", target, err)
		}
		if relToBase == ".." || strings.HasPrefix(relToBase, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("path traversal detected for %q", name)
		}

		// Create parent dirs
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", fmt.Errorf("mkdir parent for %q: %w", target, err)
		}

		data, err := toFileBytes(v)
		if err != nil {
			return "", fmt.Errorf("convert content for %q: %w", name, err)
		}

		// Write file (0644); overwrite if exists
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return "", fmt.Errorf("write %q: %w", target, err)
		}
	}

	return absDir, nil
}

func sanitizeRelativePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("empty filename")
	}

	// Clean, and normalize separators via filepath.Clean later.
	// Reject absolute paths (Unix and Windows forms).
	if filepath.IsAbs(p) {
		return "", errors.New("absolute paths are not allowed")
	}
	if vol := filepath.VolumeName(p); vol != "" {
		// e.g. "C:" on Windows
		return "", errors.New("volume paths are not allowed")
	}

	clean := filepath.Clean(p)

	// filepath.Clean can turn "." into ".", reject it as "no file".
	if clean == "." {
		return "", errors.New("invalid filename")
	}

	// Reject any traversal.
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path traversal is not allowed")
	}

	// Optional: reject paths containing NUL (can be problematic in some contexts)
	if strings.ContainsRune(clean, '\x00') {
		return "", errors.New("NUL byte in path")
	}

	return clean, nil
}

func toFileBytes(v any) ([]byte, error) {
	switch x := v.(type) {
	case nil:
		return []byte(""), nil
	case []byte:
		return x, nil
	case string:
		return []byte(x), nil
	default:
		// JSON-encode other values (maps, arrays, numbers, bools, etc.)
		b, err := json.MarshalIndent(x, "", "  ")
		if err != nil {
			return nil, err
		}
		// Add newline for nicer files
		return append(b, '\n'), nil
	}
}

func SearchPaths() []string {
	v := os.Getenv("JF_PATH")
	return strings.Split(v, ":")
}

func CreateSessionDirectory() string {
	v, ok := os.LookupEnv("JF_SESSION_DIR_BASE")
	if !ok {
		v = ""
	}
	ret, e := os.MkdirTemp(v, "jqplusplus-session-*")
	if e != nil {
		panic(fmt.Sprintf("failed to create session directory: %v", e))
	}
	return ret
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
	case ".json", "":
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
func LoadFileAsRawJSON(path string) (map[string]any, error) {
	ft, ok := detectFileType(path)
	if !ok {
		return nil, fmt.Errorf("unsupported file type: %q (%s)", filepath.Ext(path), path)
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

func readJSON(targetFileAbsPath string) (map[string]any, error) {
	data, err := os.ReadFile(targetFileAbsPath)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func readYAML(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func readTOML(path string) (map[string]any, error) {
	var m map[string]any
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
func readHOCON(path string) (map[string]any, error) {
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

func objectToMap(o hocon.Object) map[string]any {
	out := make(map[string]any, len(o))
	for k, v := range o {
		out[k] = valueToAny(v)
	}
	return out
}

func arrayToSlice(a hocon.Array) []any {
	out := make([]any, 0, len(a))
	for _, v := range a {
		out = append(out, valueToAny(v))
	}
	return out
}

func valueToAny(v hocon.Value) any {
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

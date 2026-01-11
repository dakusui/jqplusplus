package internal

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/gurkankaymak/hocon"
	"github.com/titanous/json5"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

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

// MergePolicy defines the policy for merging objects.
type MergePolicy int

const (
	MergePolicyDefault MergePolicy = iota
	// Add more policies as needed
)

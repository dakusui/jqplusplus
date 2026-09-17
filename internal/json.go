package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/gurkankaymak/hocon"
	"github.com/itchyny/gojq"
	"github.com/titanous/json5"
	"gopkg.in/yaml.v3"
)

type FileType string

const (
	JSON  FileType = "json"
	JQ    FileType = "jq"
	YAML  FileType = "yaml"
	TOML  FileType = "toml"
	JSON5 FileType = "json5"
	HCL   FileType = "hcl"
	HOCON FileType = "hocon"
)

// SupportedExtensions lists the file extensions recognized by readfile and the inheritance mechanism.
const SupportedExtensions = ".json, .json++, .jq, .yaml, .yml, .yaml++, .yml++, .toml, .toml++, .json5, .json5++, .conf, .hocon, .conf++, .hocon++ (no extension is treated as JSON)"

// InputTypeToExt maps a user-supplied input type (e.g. "yaml", ".yaml",
// "yaml++") to a canonical file-extension suffix (e.g. ".yaml") recognized by
// detectFileType. It is used to tell jq++ how to parse data read from stdin,
// which otherwise has no filename to detect the type from. It returns ok=false
// when the type is not supported.
func InputTypeToExt(t string) (string, bool) {
	ext := strings.ToLower(t)
	if ext == "" {
		return "", false
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if _, ok := detectFileType("x" + ext); ok {
		return ext, true
	}
	return "", false
}

func detectFileType(name string) (FileType, bool) {
	ext := strings.ToLower(filepath.Ext(name))

	switch ext {
	case ".json", ".json++", "":
		return JSON, true
	case ".jq":
		return JQ, true
	case ".yaml", ".yml", ".yaml++", ".yml++":
		return YAML, true
	case ".toml", ".toml++":
		return TOML, true
	case ".json5", ".json5++":
		return JSON5, true
	case ".hcl", ".hcl++":
		return HCL, true
	case ".conf", ".hocon", ".conf++", ".hocon++":
		return HOCON, true
	default:
		return "", false
	}
}

func readJSON(targetFileAbsPath string) (any, *JqModule, error) {
	data, err := os.ReadFile(targetFileAbsPath)
	if err != nil {
		return nil, nil, err
	}
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, nil, err
	}
	return obj, nil, nil
}

type moduleLoader struct {
	moduleName string
	query      *gojq.Query
}

func (l *moduleLoader) LoadModule(name string) (*gojq.Query, error) {
	if l.moduleName == name {
		return l.query, nil
	}
	return nil, fmt.Errorf("module not found: %s", name)
}

func newModuleLoader(moduleName string, moduleBody *gojq.Query) *moduleLoader {
	return &moduleLoader{moduleName, moduleBody}
}

func readJQ(targetFileAbsPath string) (any, *JqModule, error) {
	data, err := os.ReadFile(targetFileAbsPath)
	if err != nil {
		return nil, nil, err
	}
	query, err := gojq.Parse(string(data))
	if err != nil {
		return nil, nil, err
	}
	name := filepath.Base(targetFileAbsPath)
	name = strings.SplitN(filepath.Base(targetFileAbsPath), ".", 2)[0]
	ret := gojq.WithModuleLoader(newModuleLoader(name, query))
	return map[string]any{}, &JqModule{Name: name, CompilerOption: ret}, nil
}

func readYAML(path string) (any, *JqModule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, nil, err
	}
	return m, nil, nil
}

func readTOML(path string) (any, *JqModule, error) {
	var m map[string]any
	if _, err := toml.DecodeFile(path, &m); err != nil {
		return nil, nil, err
	}
	return m, nil, nil
}

func readJSON5(path string) (any, *JqModule, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var m map[string]any
	if err := json5.Unmarshal(b, &m); err != nil {
		return nil, nil, err
	}
	return m, nil, nil
}

// readHOCON reads a HOCON file and returns it as a JSON-compatible map.
// Top-level must be an object.
func readHOCON(path string) (any, *JqModule, error) {
	conf, err := hocon.ParseResource(path)
	if err != nil {
		return nil, nil, err
	}

	root := conf.GetRoot() // Value
	obj, ok := root.(hocon.Object)
	if !ok {
		return nil, nil, fmt.Errorf("HOCON top-level must be an object")
	}

	return objectToMap(obj), nil, nil
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
func mergeObjects(parent, child map[string]any) (map[string]any, error) {
	return MergeObjects(parent, child, MergePolicyDefault)
}

func MergeObjects(a, b map[string]interface{}, policy MergePolicy) (map[string]interface{}, error) {
	return mergeObjectsAt(a, b, policy, nil)
}

func mergeObjectsAt(a, b map[string]interface{}, policy MergePolicy, path []any) (map[string]interface{}, error) {
	// childPath copies rather than appending in place: sibling keys would
	// otherwise share a backing array and overwrite each other's last segment.
	childPath := func(k string) []any {
		p := make([]any, len(path), len(path)+1)
		copy(p, path)
		return append(p, k)
	}
	result := make(map[string]interface{})
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		if av, ok := result[k].(map[string]interface{}); ok {
			if bv, ok := v.(map[string]interface{}); ok {
				merged, err := mergeObjectsAt(av, bv, policy, childPath(k))
				if err != nil {
					return nil, err
				}
				result[k] = merged
				continue
			}
		}
		if child, ok := v.([]any); ok {
			inherited, present := result[k]
			composed, err := composeArrayAt(inherited, present, child, childPath(k))
			if err != nil {
				return nil, err
			}
			result[k] = composed
			continue
		}
		result[k] = v
	}
	return result, nil
}

// composeArrayAt composes a child array with the array it inherits. An unmarked
// array replaces, exactly as it always has. A marked array is a delta: it
// carries unchanged while the inherited array is absent, and resolves against
// it once one is present.
func composeArrayAt(inherited any, present bool, child []any, path []any) (any, error) {
	kind, at, err := scanArrayMarkers(child)
	if err != nil {
		return nil, fmt.Errorf("%v: at '%s'", err, toPathExpression(path))
	}
	if !kind.IsMarker() {
		return child, nil
	}
	if !present {
		// Nothing to bind to yet. The delta carries, which is what lets a
		// fragment resolved on its own contribute to a document that includes
		// it later.
		return child, nil
	}
	inheritedArray, ok := inherited.([]any)
	if !ok {
		return nil, fmt.Errorf("array composition marker requires an inherited array, but got %T: at '%s'", inherited, toPathExpression(path))
	}
	switch kind {
	case SpliceMarker:
		return spliceArray(inheritedArray, child, at), nil
	default:
		// MarkerPair is not implemented yet; it carries and is reported by
		// grounding until pairing lands.
		return child, nil
	}
}

// spliceArray substitutes the inherited elements at the marker's position,
// building a fresh slice: the node pool shares backing arrays between every
// document that inherits a file, so appending into one would corrupt them all.
//
// Inherited elements are copied verbatim, markers included. That is what makes
// two splice deltas compose into a third, still-pending delta.
func spliceArray(inherited []any, child []any, at int) []any {
	out := make([]any, 0, len(child)-1+len(inherited))
	out = append(out, child[:at]...)
	out = append(out, inherited...)
	out = append(out, child[at+1:]...)
	return out
}

// scanArrayMarkers finds the array's marker, rejecting an undefined spelling in
// the reserved namespace and an array carrying more than one marker.
func scanArrayMarkers(arr []any) (MarkerKind, int, error) {
	kind, at := NotAMarker, -1
	for i, e := range arr {
		k, err := ClassifyMarkerElement(e)
		if err != nil {
			return NotAMarker, -1, err
		}
		if !k.IsMarker() {
			continue
		}
		if at >= 0 {
			return NotAMarker, -1, fmt.Errorf("more than one array composition marker: %v", arr)
		}
		kind, at = k, i
	}
	return kind, at, nil
}

// MergePolicy defines the policy for merging objects.
type MergePolicy int

const (
	MergePolicyDefault MergePolicy = iota
	// Add more policies as needed
)

// PathArrayToPathExpression converts a "path array" to a "path expression" string.
func PathArrayToPathExpression(pathArray []any) (string, error) {
	var result string

	for _, elem := range pathArray {
		switch v := elem.(type) {
		case string:
			// Handle alphanumeric keys directly, quote and escape non-alphanumerical keys
			if isAlphanumeric(v) {
				result += fmt.Sprintf(".%s", v)
			} else {
				// Quote keys if they contain special characters
				escaped := escapeString(v)
				result += fmt.Sprintf("[\"%s\"]", escaped)
			}
		case int: // Array index
			result += fmt.Sprintf("[%d]", v)
		default:
			return "", fmt.Errorf("unsupported path array element type: %T", v)
		}
	}

	// Return the constructed path expression
	return result, nil
}

// PathExpressionToPathArray converts a "path expression" string to a "path array".
func PathExpressionToPathArray(pathExpr string) ([]any, error) {
	var result []any
	var buffer string
	inBracket := false
	inQuotes := false
	escaped := false

	for i, r := range pathExpr {
		switch {
		case escaped:
			// Handle escaped characters
			buffer += string(r)
			escaped = false

		case r == '\\':
			// Mark next character as escaped
			escaped = true

		case inQuotes:
			// Inside quoted strings
			if r == '"' {
				// End quote
				inQuotes = false
				result = append(result, buffer)
				buffer = ""
			} else {
				buffer += string(r)
			}

		case inBracket:
			// Inside brackets
			if r == ']' {
				// End of bracket. Convert buffer contents.
				inBracket = false
				if num, err := strconv.Atoi(buffer); err == nil {
					result = append(result, num)
				} else {
					return nil, fmt.Errorf("invalid array index at position %d: %s", i, buffer)
				}
				buffer = ""
			} else {
				buffer += string(r)
			}

		default:
			// Outside quotes/brackets
			switch r {
			case '.':
				// Period starts a new segment if buffer has content
				if buffer != "" {
					result = append(result, buffer)
					buffer = ""
				}
			case '[':
				// Start of bracket
				if buffer != "" {
					result = append(result, buffer)
					buffer = ""
				}
				inBracket = true
			case '"':
				// Start of quoted string
				inQuotes = true
			default:
				buffer += string(r)
			}
		}
	}

	// Add remaining buffer content to result
	if buffer != "" {
		result = append(result, buffer)
	}

	// Return the constructed path array
	return result, nil
}

// Helper to check if a string is alphanumeric
func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return false
		}
	}
	return true
}

// Helper to escape a string for use in a path expression
func escapeString(s string) string {
	escaped := ""
	for _, r := range s {
		switch r {
		case '\\':
			escaped += `\\`
		case '"':
			escaped += `\"`
		default:
			escaped += string(r)
		}
	}
	return escaped
}

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
func mergeObjects(parent, child map[string]any) map[string]any {
	return MergeObjects(parent, child, MergePolicyDefault)
}

// mergeObjectsForInheritance merges a parent object with a child object and
// resolves array-composition markers that occur in the child.
func mergeObjectsForInheritance(parent, child map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(parent)+len(child))
	for k, v := range parent {
		result[k] = v
	}
	for k, childValue := range child {
		parentValue, exists := result[k]
		if parentObject, ok := parentValue.(map[string]any); ok {
			if childObject, ok := childValue.(map[string]any); ok {
				merged, err := mergeObjectsForInheritance(parentObject, childObject)
				if err != nil {
					return nil, err
				}
				result[k] = merged
				continue
			}
		}
		if childArray, ok := childValue.([]any); ok && hasArrayMarker(childArray) {
			if !exists {
				// A delta carries until a later inheritance step provides a value.
				result[k] = childArray
				continue
			}
			parentArray, ok := parentValue.([]any)
			if !ok {
				return nil, fmt.Errorf("%s: inherited value must be an array for array composition", k)
			}
			merged, err := mergeArrayComposition(parentArray, childArray)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			result[k] = merged
			continue
		}
		result[k] = childValue
	}
	return result, nil
}

func containsMarker(values []any, marker string) bool {
	for _, value := range values {
		if value == marker {
			return true
		}
	}
	return false
}

func hasArrayMarker(values []any) bool {
	return containsMarker(values, "$super") || containsMarker(values, "$super*")
}

func mergeArrayComposition(inherited, child []any) ([]any, error) {
	hasSuper := containsMarker(child, "$super")
	starCount := 0
	for _, value := range child {
		if value == "$super*" {
			starCount++
		}
	}
	if hasSuper && starCount > 0 {
		return nil, fmt.Errorf("cannot mix $super and $super* in one array")
	}
	if starCount > 1 {
		return nil, fmt.Errorf("$super* may appear only once in an array")
	}
	if hasSuper {
		if hasArrayMarker(inherited) {
			if containsMarker(inherited, "$super*") {
				return nil, fmt.Errorf("cannot compose $super with $super*")
			}
		}
		return spliceSuper(inherited, child), nil
	}
	if starCount == 1 {
		if hasArrayMarker(inherited) {
			return nil, fmt.Errorf("cannot compose $super* with a marked array")
		}
		return pairSuper(inherited, child)
	}
	return child, nil
}

func spliceSuper(inherited, child []any) []any {
	result := make([]any, 0, len(inherited)+len(child)-1)
	for _, value := range child {
		if value == "$super" {
			result = append(result, inherited...)
			continue
		}
		result = append(result, value)
	}
	return result
}

func pairSuper(inherited, child []any) ([]any, error) {
	markerIndex := 0
	for child[markerIndex] != "$super*" {
		markerIndex++
	}
	prefix := child[:markerIndex]
	queue := child[markerIndex+1:]
	result := make([]any, 0, len(prefix)+max(len(inherited), len(queue)))
	result = append(result, prefix...)
	paired := min(len(inherited), len(queue))
	for i := 0; i < paired; i++ {
		value, err := mergeArrayPair(inherited[i], queue[i])
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	result = append(result, inherited[paired:]...)
	result = append(result, queue[paired:]...)
	return result, nil
}

func mergeArrayPair(inherited, queued any) (any, error) {
	switch inherited := inherited.(type) {
	case map[string]any:
		queuedObject, ok := queued.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cross-kind array pair: object and %s", valueKind(queued))
		}
		return mergeObjectsForInheritance(inherited, queuedObject)
	case []any:
		queuedArray, ok := queued.([]any)
		if !ok {
			return nil, fmt.Errorf("cross-kind array pair: array and %s", valueKind(queued))
		}
		if hasArrayMarker(queuedArray) {
			return mergeArrayComposition(inherited, queuedArray)
		}
		return queuedArray, nil
	default:
		switch queued.(type) {
		case map[string]any, []any:
			return nil, fmt.Errorf("cross-kind array pair: atom and %s", valueKind(queued))
		default:
			return queued, nil
		}
	}
}

func valueKind(value any) string {
	switch value.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return "atom"
	}
}

// GroundArrayMarkers verifies that all composition markers in a final document
// have either been resolved or escaped with raw:. It must be called only for
// the top-level rendered document, never for cached inherited fragments.
func GroundArrayMarkers(obj map[string]any) error {
	return groundObject(obj)
}

func groundObject(obj map[string]any) error {
	for key, value := range obj {
		if err := validateMarkerString(key, false); err != nil {
			return err
		}
		if err := groundValue(value, false); err != nil {
			return err
		}
	}
	return nil
}

func groundValue(value any, nestedArray bool) error {
	switch value := value.(type) {
	case map[string]any:
		return groundObject(value)
	case []any:
		for _, element := range value {
			if stringValue, ok := element.(string); ok {
				if err := validateMarkerString(stringValue, !nestedArray); err != nil {
					return err
				}
				if stringValue == "$super" || stringValue == "$super*" {
					return fmt.Errorf("unresolved marker %q", stringValue)
				}
			}
			if err := groundValue(element, true); err != nil {
				return err
			}
		}
	case string:
		return validateMarkerString(value, false)
	}
	return nil
}

func validateMarkerString(value string, arrayElement bool) error {
	if strings.HasPrefix(value, "raw:") {
		return nil
	}
	if value == "$super" || value == "$super*" {
		if !arrayElement {
			return fmt.Errorf("out-of-context marker %q", value)
		}
		return nil
	}
	if strings.HasPrefix(value, "$super") {
		suffix := value[len("$super"):]
		if suffix != "" && !startsIdentifier(suffix) {
			return fmt.Errorf("unknown marker %q", value)
		}
	}
	return nil
}

func startsIdentifier(value string) bool {
	for _, r := range value {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
	}
	return false
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

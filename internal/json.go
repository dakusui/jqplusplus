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

func mergeObjectsForInheritance(parent, child map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(parent)+len(child))
	for k, v := range parent {
		result[k] = v
	}
	for k, childValue := range child {
		parentValue, exists := result[k]
		if !exists {
			result[k] = childValue
			continue
		}
		merged, err := mergeInheritanceValues(parentValue, childValue)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", k, err)
		}
		result[k] = merged
	}
	return result, nil
}

func mergeInheritanceValues(parent, child any) (any, error) {
	parentObject, parentIsObject := parent.(map[string]any)
	childObject, childIsObject := child.(map[string]any)
	if parentIsObject && childIsObject {
		return mergeObjectsForInheritance(parentObject, childObject)
	}
	childArray, childIsArray := child.([]any)
	if !childIsArray {
		return child, nil
	}
	markers := arrayMarkers(childArray)
	if !markers.hasSuper && markers.superStarCount == 0 {
		return child, nil
	}
	if markers.hasSuper && markers.superStarCount != 0 {
		return nil, fmt.Errorf("cannot mix $super and $super* in the same array")
	}
	if markers.superStarCount > 1 {
		return nil, fmt.Errorf("$super* may appear only once in an array")
	}
	parentArray, parentIsArray := parent.([]any)
	if !parentIsArray {
		return nil, fmt.Errorf("array marker requires an inherited array, got %s", valueKind(parent))
	}
	parentMarkers := arrayMarkers(parentArray)
	if markers.superStarCount != 0 && (parentMarkers.hasSuper || parentMarkers.superStarCount != 0) {
		return nil, fmt.Errorf("cannot compose $super* with a marked inherited array")
	}
	if markers.superStarCount != 0 {
		return pairSuper(parentArray, childArray, markers.superStarIndex)
	}
	return spliceSuper(parentArray, childArray)
}

type superMarkers struct {
	hasSuper       bool
	superStarCount int
	superStarIndex int
}

func arrayMarkers(array []any) superMarkers {
	markers := superMarkers{superStarIndex: -1}
	for index, value := range array {
		switch value {
		case "$super":
			markers.hasSuper = true
		case "$super*":
			markers.superStarCount++
			markers.superStarIndex = index
		}
	}
	return markers
}

func spliceSuper(parent, child []any) ([]any, error) {
	result := make([]any, 0, len(parent)+len(child))
	for _, value := range child {
		if value == "$super" {
			result = append(result, parent...)
			continue
		}
		result = append(result, value)
	}
	return result, nil
}

func pairSuper(parent, child []any, markerIndex int) ([]any, error) {
	prefix := child[:markerIndex]
	queue := child[markerIndex+1:]
	result := make([]any, 0, len(prefix)+max(len(parent), len(queue)))
	result = append(result, prefix...)
	pairedLength := min(len(parent), len(queue))
	for index := 0; index < pairedLength; index++ {
		merged, err := mergePairedValues(parent[index], queue[index])
		if err != nil {
			return nil, fmt.Errorf("$super* pair %d: %w", index, err)
		}
		result = append(result, merged)
	}
	result = append(result, parent[pairedLength:]...)
	result = append(result, queue[pairedLength:]...)
	return result, nil
}

func mergePairedValues(parent, child any) (any, error) {
	parentKind, childKind := valueKind(parent), valueKind(child)
	if parentKind != childKind {
		return nil, fmt.Errorf("cannot pair %s with %s", parentKind, childKind)
	}
	switch parentKind {
	case "object":
		return mergeObjectsForInheritance(parent.(map[string]any), child.(map[string]any))
	case "array":
		return mergeInheritanceValues(parent, child)
	default:
		return child, nil
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

// ValidateMarkersAreGrounded validates the reserved $super namespace on the
// rendered top-level object. Cached inheritance fragments deliberately bypass
// this check so that an unresolved delta can be composed by its consumer.
func ValidateMarkersAreGrounded(obj map[string]any) error {
	return validateMarkerValue(obj, false, false)
}

func validateMarkerValue(value any, directArrayElement, markerArray bool) error {
	switch x := value.(type) {
	case map[string]any:
		for key, child := range x {
			if err := validateMarkerString(key, false); err != nil {
				return err
			}
			if err := validateMarkerValue(child, false, true); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range x {
			if err := validateMarkerValue(child, true, markerArray); err != nil {
				return err
			}
		}
	case string:
		if err := validateMarkerString(x, directArrayElement && markerArray); err != nil {
			return err
		}
	}
	return nil
}

func validateMarkerString(value string, validMarkerPosition bool) error {
	if strings.HasPrefix(value, "raw:") || strings.HasPrefix(value, "eval:") {
		return nil
	}
	if value == "$super" || value == "$super*" {
		if validMarkerPosition {
			return fmt.Errorf("unresolved marker %q", value)
		}
		return fmt.Errorf("marker %q is out of context", value)
	}
	if strings.HasPrefix(value, "$super") {
		rest := value[len("$super"):]
		if rest != "" {
			r, _ := utf8DecodeRuneInString(rest)
			if !isMarkerIdentifier(r) {
				return fmt.Errorf("unknown marker %q", value)
			}
		}
	}
	return nil
}

func isMarkerIdentifier(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

func utf8DecodeRuneInString(value string) (rune, int) {
	for _, r := range value {
		return r, len(string(r))
	}
	return 0, 0
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

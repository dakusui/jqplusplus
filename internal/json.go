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

// mergeObjects merges parent and child objects during inheritance resolution.
// Array composition is opt-in at each child array through $super/$super*.
func mergeObjects(parent, child map[string]any) map[string]any {
	result, err := mergeObjectsWithError(parent, child)
	if err != nil {
		panic(err)
	}
	return result
}

func mergeObjectsWithError(parent, child map[string]any) (map[string]any, error) {
	return mergeObjectsWithPolicy(parent, child, MergePolicyInheritance)
}

// MergeObjects recursively merges object values. With the default policy,
// arrays are replaced wholesale, preserving jq++'s historical behavior. The
// inheritance policy additionally recognizes the explicit array markers.
//
// MergeObjects keeps its original no-error API for callers that use the
// ordinary merge policy. Invalid explicit inheritance operations are reported
// by the inheritance resolver through mergeObjectsWithPolicy instead.
func MergeObjects(a, b map[string]interface{}, policy MergePolicy) map[string]interface{} {
	result, err := mergeObjectsWithPolicy(a, b, policy)
	if err != nil {
		panic(err)
	}
	return result
}

func mergeObjectsWithPolicy(parent, child map[string]any, policy MergePolicy) (map[string]any, error) {
	result := make(map[string]any, len(parent)+len(child))
	for k, v := range parent {
		result[k] = v
	}
	for k, v := range child {
		pv, exists := result[k]
		if !exists {
			if policy == MergePolicyInheritance {
				result[k] = copyMarkedArray(v)
			} else {
				result[k] = v
			}
			continue
		}

		merged, err := mergeValuesWithPolicy(pv, v, policy)
		if err != nil {
			return nil, err
		}
		result[k] = merged
	}
	return result, nil
}

// MergePolicy defines the policy for merging objects.
type MergePolicy int

const (
	// MergePolicyDefault replaces arrays wholesale.
	MergePolicyDefault MergePolicy = iota
	// MergePolicyInheritance enables explicit $super/$super* array operations.
	MergePolicyInheritance
)

const (
	arrayMarkerNone = iota
	arrayMarkerSplice
	arrayMarkerPairwise
)

const (
	arraySpliceToken = "$super"
	arrayPairToken   = "$super*"
)

type arrayMarkerInfo struct {
	kind  int
	index int
}

func inspectArrayMarker(array []any) (arrayMarkerInfo, error) {
	info := arrayMarkerInfo{kind: arrayMarkerNone, index: -1}
	for i, value := range array {
		marker, ok := value.(string)
		if !ok {
			continue
		}
		switch marker {
		case arraySpliceToken:
			if info.kind == arrayMarkerPairwise {
				return arrayMarkerInfo{}, fmt.Errorf("array cannot mix %q and %q", arraySpliceToken, arrayPairToken)
			}
			info.kind = arrayMarkerSplice
		case arrayPairToken:
			if info.kind == arrayMarkerSplice {
				return arrayMarkerInfo{}, fmt.Errorf("array cannot mix %q and %q", arraySpliceToken, arrayPairToken)
			}
			if info.kind == arrayMarkerPairwise {
				return arrayMarkerInfo{}, fmt.Errorf("array can contain only one %q marker", arrayPairToken)
			}
			info.kind = arrayMarkerPairwise
			info.index = i
		}
	}
	return info, nil
}

func copyMarkedArray(value any) any {
	array, ok := value.([]any)
	if !ok {
		return value
	}
	info, err := inspectArrayMarker(array)
	if err != nil {
		return value
	}
	if info.kind == arrayMarkerNone {
		return value
	}
	return DeepCopy(array)
}

func mergeValuesWithPolicy(parent, child any, policy MergePolicy) (any, error) {
	parentObject, parentIsObject := parent.(map[string]any)
	childObject, childIsObject := child.(map[string]any)
	if parentIsObject && childIsObject {
		return mergeObjectsWithPolicy(parentObject, childObject, policy)
	}

	parentArray, parentIsArray := parent.([]any)
	childArray, childIsArray := child.([]any)
	if parentIsArray && childIsArray {
		if policy != MergePolicyInheritance {
			return child, nil
		}
		return mergeInheritanceArrays(parentArray, childArray)
	}
	if policy == MergePolicyInheritance && childIsArray {
		childInfo, err := inspectArrayMarker(childArray)
		if err != nil {
			return nil, err
		}
		if childInfo.kind != arrayMarkerNone {
			return nil, fmt.Errorf("explicit array merge requires an inherited array, got %s", jsonMergeKind(parent))
		}
	}
	return child, nil
}

func mergeInheritanceArrays(parent, child []any) ([]any, error) {
	childInfo, err := inspectArrayMarker(child)
	if err != nil {
		return nil, err
	}
	if childInfo.kind == arrayMarkerNone {
		// The marker is opt-in at the child site; an unmarked child array keeps
		// the original replacement behavior even when the parent was marked.
		return child, nil
	}

	parentInfo, err := inspectArrayMarker(parent)
	if err != nil {
		return nil, err
	}
	if childInfo.kind == arrayMarkerPairwise && parentInfo.kind != arrayMarkerNone {
		return nil, fmt.Errorf("marked arrays cannot be merged: parent and child both contain an explicit array marker")
	}
	if childInfo.kind == arrayMarkerSplice && parentInfo.kind == arrayMarkerPairwise {
		return nil, fmt.Errorf("marked arrays cannot be merged: $super cannot compose with $super*")
	}

	switch childInfo.kind {
	case arrayMarkerSplice:
		result := make([]any, 0, len(parent)+len(child)-1)
		for _, value := range child {
			if marker, ok := value.(string); ok && marker == arraySpliceToken {
				for _, inherited := range parent {
					result = append(result, DeepCopy(inherited))
				}
				continue
			}
			result = append(result, DeepCopy(value))
		}
		return result, nil
	case arrayMarkerPairwise:
		prefix := child[:childInfo.index]
		queue := child[childInfo.index+1:]
		result := make([]any, 0, len(prefix)+maxInt(len(parent), len(queue)))
		for _, value := range prefix {
			result = append(result, DeepCopy(value))
		}
		paired := minInt(len(parent), len(queue))
		for i := 0; i < paired; i++ {
			merged, err := mergePairwiseValues(parent[i], queue[i])
			if err != nil {
				return nil, fmt.Errorf("array pair at index %d: %w", i, err)
			}
			result = append(result, merged)
		}
		for _, value := range parent[paired:] {
			result = append(result, DeepCopy(value))
		}
		for _, value := range queue[paired:] {
			result = append(result, DeepCopy(value))
		}
		return result, nil
	default:
		panic("unknown array marker")
	}
}

func mergePairwiseValues(parent, child any) (any, error) {
	parentObject, parentIsObject := parent.(map[string]any)
	childObject, childIsObject := child.(map[string]any)
	if parentIsObject && childIsObject {
		// Strictness applies to the paired values themselves. Once an object
		// pair is entered, its fields use the language's ordinary merge rules.
		return mergeObjectsWithPolicy(parentObject, childObject, MergePolicyInheritance)
	}

	parentArray, parentIsArray := parent.([]any)
	childArray, childIsArray := child.([]any)
	if parentIsArray && childIsArray {
		childInfo, err := inspectArrayMarker(childArray)
		if err != nil {
			return nil, err
		}
		if childInfo.kind == arrayMarkerNone {
			return DeepCopy(childArray), nil
		}
		return mergeInheritanceArrays(parentArray, childArray)
	}

	parentKind := jsonMergeKind(parent)
	childKind := jsonMergeKind(child)
	if parentKind != childKind {
		return nil, fmt.Errorf("cannot merge %s with %s", parentKind, childKind)
	}
	// Atom pairs are deliberately child-wins, including null.
	return DeepCopy(child), nil
}

func jsonMergeKind(value any) string {
	switch value.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return "atom"
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

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

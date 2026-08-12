package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LoadAndResolveInheritances loads a JSON file, resolves filelevel, and returns the merged result as a map.
func LoadAndResolveInheritances(baseDir string, filename string, searchPaths []string) (*NodeEntryValue, error) {
	sessionDirectory := CreateSessionDirectory()
	defer func() {
		err := os.RemoveAll(CreateSessionDirectory())
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("failed to remove directory: %s", err))
		}
	}()

	nodeEntryValue, err := NewNodePoolWithBaseSearchPaths(baseDir, sessionDirectory, searchPaths).ReadNodeEntryValue(baseDir, filename, []*JqModule{})
	if err != nil {
		return nil, err
	}
	// Final validation sweep: all inheritance is settled, so any array
	// composition token still present is dangling merge intent, and unknown
	// tokens in the reserved namespace are reported here as well.
	if err := ValidateArrayCompositions(nodeEntryValue.Obj); err != nil {
		return nil, err
	}
	return nodeEntryValue, nil
}

// LoadAndResolveInheritancesRecursively loads a JSON file, resolves $extends or $includes recursively, and merges parents.
func LoadAndResolveInheritancesRecursively(baseDir string, targetFile string, nodepool NodePool) (*NodeEntryValue, error) {
	var optional bool
	if strings.HasSuffix(targetFile, "?") {
		optional = true
		targetFile = targetFile[:len(targetFile)-1]
	}
	absPath, err := ResolveFilePath(targetFile, baseDir, nodepool.SearchPaths())
	if optional && errors.Is(err, fs.ErrNotExist) {
		return &NodeEntryValue{Obj: map[string]any{}, CompilerOptions: make([]*JqModule, 0)}, nil
	}
	if err != nil {
		return nil, composeFileNotFoundError(targetFile)
	}

	// From this point on, inheritance resolution must be anchored at the
	// actual resolved file location, not the caller's baseDir.
	resolvedBaseDir := filepath.Dir(absPath)

	if nodepool.IsVisited(absPath) {
		visitedFiles := append(nodepool.VisitedFiles(absPath), absPath)
		circulatingFiles := formatCirculatingFileLoop(visitedFiles, resolvedBaseDir)
		return nil, fmt.Errorf("circular inheritance detected: [%s]", circulatingFiles)
	}
	nodepool.MarkVisited(absPath)

	obj, compilerOption, err := ReadFileAsJSONObject(absPath)
	if err != nil {
		return nil, err
	}

	{
		// Materialize local nodes
		localNodeDirectoryBase := nodepool.SessionDirectory()
		absDir := filepath.Join(localNodeDirectoryBase, baseDir, targetFile)
		err := os.MkdirAll(absDir, 0o755)
		if err != nil {
			return nil, fmt.Errorf("failed: mkdir temp dir: %w", err)
		}
		nodepool.Enter(absDir)
		_, err = MaterializeLocalNodes(obj, nodepool.LocalNodeDirectory())
		if err != nil {
			return nil, err
		}
		delete(obj, "$local")
	}

	var nodeEntryValue *NodeEntryValue
	{
		// File-level inheritance
		slog.Debug("BEGIN: File-level inheritance: ", "targetFile", targetFile)
		nodeEntryValue, err = expandFileLevelInheritances(obj, compilerOptions(compilerOption), nodepool, resolvedBaseDir)
		if err != nil {
			return nil, composeInheritanceResolutionErr(err, "file")
		}
		slog.Debug("END:   File-level inheritance: ", "targetFile", targetFile)
	}
	{
		// Node-level inheritance
		nodeEntryValue, err = expandNodeLevelInheritances(nodeEntryValue.Obj, nodeEntryValue.CompilerOptions, nodepool, resolvedBaseDir)
		if err != nil {
			return nil, composeInheritanceResolutionErr(err, "node")
		}
	}
	return nodeEntryValue, nil
}
func composeFileNotFoundError(filename string) error {
	return fmt.Errorf("file not found: %q", filename)
}

func compilerOptions(compilerOption *JqModule) []*JqModule {
	if compilerOption == nil {
		return []*JqModule{}
	}
	return []*JqModule{compilerOption}
}

func expandFileLevelInheritances(obj map[string]any, compilerOptions []*JqModule, nodepool NodePool, baseDir string) (*NodeEntryValue, error) {
	nodeEntryValue, err := resolveBothInheritances(obj, compilerOptions, nodepool, baseDir)
	return nodeEntryValue, err
}

func expandNodeLevelInheritances(obj map[string]any, compilerOptions []*JqModule, nodepool NodePool, baseDir string) (*NodeEntryValue, error) {
	for _, nodepath := range DistinctBy(Map(Sort(Paths(obj, lastElementIsOneOf("$extends", "$includes")), lessPathArrays), DropLast[any]), pathKey) {
		slog.Debug("BEGIN: Node-level inheritance: ", "nodepath", nodepath)
		internal, ok := GetAtPath(obj, ToAnySlice(nodepath))
		if !ok {
			continue
		}
		internalObj, ok := internal.(map[string]any)
		if !ok {
			continue
		}
		nodeEntryValue, err := resolveBothInheritances(internalObj, compilerOptions, nodepool, baseDir)
		if err != nil {
			np := toPathExpression(nodepath)
			return nil, fmt.Errorf("%s: at '%v'", err, np)
		}
		internalObj = nodeEntryValue.Obj
		compilerOptions = nodeEntryValue.CompilerOptions
		PutAtPath(obj, ToAnySlice(nodepath), internalObj)
		slog.Debug("END: Node-level inheritance: ", "nodepath", nodepath)
	}
	return &NodeEntryValue{Obj: obj, CompilerOptions: compilerOptions}, nil
}

func resolveBothInheritances(obj map[string]any, compilerOptions []*JqModule, nodepool NodePool, baseDir string) (*NodeEntryValue, error) {
	ret := &NodeEntryValue{Obj: obj, CompilerOptions: compilerOptions}
	ret, err := resolveInheritances(ret.Obj, ret.CompilerOptions, nodepool, baseDir, Extends)
	if err != nil {
		return nil, err
	}
	ret, err = resolveInheritances(ret.Obj, ret.CompilerOptions, nodepool, baseDir, Includes)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func resolveInheritances(obj map[string]any, compilerOptions []*JqModule, nodepool NodePool, baseDir string, mergeType InheritType) (*NodeEntryValue, error) {
	tmpCompilerOptions := compilerOptions
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
			nodeEntryValue, err := nodepool.ReadNodeEntryValue(baseDir, parent, tmpCompilerOptions)
			if err != nil {
				return nil, fmt.Errorf("%v: parent: %v", err, parent)
			}
			if i == 0 {
				mergedParents = nodeEntryValue.Obj
			} else {
				mergedParents, err = MergeNodes(mergedParents, nodeEntryValue.Obj)
				if err != nil {
					return nil, fmt.Errorf("%v: parent: %v", err, parent)
				}
			}
			tmpCompilerOptions = append(tmpCompilerOptions, nodeEntryValue.CompilerOptions...)
		}
		if !mergeType.IsOrderReversed() {
			obj, err = MergeNodes(mergedParents, obj)
		} else {
			obj, err = MergeNodes(obj, mergedParents)
		}
		if err != nil {
			return nil, err
		}
		delete(obj, mergeType.String())
	}
	return &NodeEntryValue{Obj: obj, CompilerOptions: tmpCompilerOptions}, nil
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
			result = Insert(result, 0, str)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%s must be an array of strings", inherits.String())
	}
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

// ReadFileAsJSONObject loads and parses a file (JSON, YAML, etc.) into a gojq-compatible object.
func ReadFileAsJSONObject(path string) (map[string]any, *JqModule, error) {
	ret, jqModuel, err := ReadFileAsJSONElement(path)
	if err != nil {
		return nil, nil, fmt.Errorf("%v: failed to read file: %s", err, path)
	}
	m, ok := ret.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("expected JSON object, but got %T(%v): in %s", ret, ret, path)
	}
	return m, jqModuel, nil
}

func ReadFileAsJSONElement(path string) (any, *JqModule, error) {
	ft, ok := detectFileType(path)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported file type: %q (%s); supported extensions are: %s", filepath.Ext(path), path, SupportedExtensions)
	}

	switch ft {
	case JSON:
		return readJSON(path)
	case JQ:
		return readJQ(path)
	case YAML:
		return readYAML(path)
	case TOML:
		return readTOML(path)
	case JSON5:
		return readJSON5(path)
	case HCL:
		return nil, nil, fmt.Errorf("unsupported file type: %q (%s)", ft, path)
	case HOCON:
		return readHOCON(path)
	default:
		return nil, nil, fmt.Errorf("unsupported file type: %q (%s)", ft, path)
	}
}

func lastElementIsOneOf(v ...string) func(p []any) bool {
	return func(p []any) bool {
		if len(p) == 0 {
			return false
		}
		s, ok := p[len(p)-1].(string)
		if !ok {
			return false
		}
		for _, v := range v {
			if s == v {
				return true
			}
		}
		return false
	}
}

func lessPathArrays(a []any, b []any) bool {
	pea, e := PathArrayToPathExpression(a)
	if e != nil {
		panic(e)
	}
	peb, e := PathArrayToPathExpression(b)
	if e != nil {
		panic(e)
	}
	return pea > peb
}

func pathKey(p []any) string {
	var b strings.Builder
	for _, v := range p {
		switch x := v.(type) {
		case string:
			b.WriteString("s:")
			b.WriteString(x)
		case int:
			b.WriteString("i:")
			b.WriteString(strconv.Itoa(x))
		default:
			panic("unsupported type in path")
		}
		b.WriteByte('|')
	}
	return b.String()
}

func toPathExpression(nodepath []any) string {
	np, err := PathArrayToPathExpression(nodepath)
	if err != nil {
		return fmt.Sprintf("%v", nodepath)
	}
	return np
}

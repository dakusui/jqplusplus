package internal

import (
	"fmt"
	"github.com/itchyny/gojq"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LoadAndResolveInheritances loads a JSON file, resolves filelevel, and returns the merged result as a map.
func LoadAndResolveInheritances(baseDir string, filename string, searchPaths []string) (map[string]any, error) {
	sessionDirectory := CreateSessionDirectory()
	defer func() {
		err := os.RemoveAll(CreateSessionDirectory())
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("failed to remove directory: %s", err))
		}
	}()

	return NewNodePoolWithBaseSearchPaths(baseDir, sessionDirectory, searchPaths).ReadNodeEntry(baseDir, filename)
}

// LoadAndResolveInheritancesRecursively loads a JSON file, resolves $extends or $includes recursively, and merges parents.
func LoadAndResolveInheritancesRecursively(baseDir string, targetFile string, nodepool NodePool) (map[string]any, error) {
	absPath, bDir, err := ResolveFilePath(targetFile, baseDir, nodepool.SearchPaths())
	if err != nil {
		return nil, err
	}
	if nodepool.IsVisited(absPath) {
		return nil, fmt.Errorf("circular filelevel inheritance detected: %s", absPath)
	}
	nodepool.MarkVisited(absPath)

	obj, _, err := LoadFileAsRawJSON(absPath)
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

	for _, p := range DistinctBy(Map(Sort(Paths(obj, lastElementIsOneOf("$extends", "$includes")), lessPathArrays), DropLast[any]), pathKey) {
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
		PutAtPath(obj, ToAnySlice(p), internalObj)
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

// LoadFileAsRawJSON loads and parses a file (JSON, YAML, etc.) into a gojq-compatible object.
func LoadFileAsRawJSON(path string) (map[string]any, gojq.CompilerOption, error) {
	ft, ok := detectFileType(path)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported file type: %q (%s)", filepath.Ext(path), path)
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
	return pea < peb
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

package internal

import (
	"encoding/json"
	"fmt"
)

// CreateToPathArrayFunc returns a builtin function descriptor for "topatharray".
// The builtin takes one argument: a path expression string (e.g. "$.foo.bar[0]").
// It returns the path as a slice of path segments (path array), or an error if
// the expression is invalid. The returned tuple is (name, minArgs, maxArgs, impl).
func CreateToPathArrayFunc() (string, int, int, func(any, []any) any) {
	return "topatharray", 1, 1, func(input any, args []any) any {
		ret, err := PathExpressionToPathArray(args[0].(string))
		if err != nil {
			return err
		}
		return ret
	}
}

// CreateToPathExprFunc returns a builtin function descriptor for "topathexpr".
// The builtin takes one argument: a path array (slice of path segments).
// It returns the path as an expression string (e.g. "$.foo.bar[0]"), or an error
// if the path is invalid. The returned tuple is (name, minArgs, maxArgs, impl).
func CreateToPathExprFunc() (string, int, int, func(any, []any) any) {
	return "topathexpr", 1, 1, func(input any, args []any) any {
		ret, err := PathArrayToPathExpression(args[0].([]any))
		if err != nil {
			return err
		}
		return ret
	}
}

// CreateParentOfFunc returns a builtin function descriptor for "parentof".
// The builtin takes a path array and an optional level count (default 1), and
// returns the path with the last level(s) removed. currentPath and expression
// are used for error reporting. The returned tuple is (name, minArgs, maxArgs, impl).
func CreateParentOfFunc(currentPath []any, expression string) (string, int, int, func(any, []any) any) {
	return "parentof", 1, 2, func(input any, args []any) any {
		return resolveParent(args, expression, currentPath)
	}
}

// CreateParentFunc returns a builtin function descriptor for "parent".
// The builtin takes zero or one argument (parent level count; default 1) and
// returns the parent of the current path. currentPath and expression are used
// for error reporting. The returned tuple is (name, minArgs, maxArgs, impl).
func CreateParentFunc(currentPath []any, expression string) (string, int, int, func(any, []any) any) {
	return "parent", 0, 1, func(input any, args []any) any {
		return resolveParent(append([]any{currentPath}, args...), expression, currentPath)
	}
}

// resolveParent is the shared implementation for "parent" and "parentof".
// args[0] must be a path array; args[1] if present must be the number of
// levels to go up. Returns the path with the last levels segments removed, or an error.
func resolveParent(args []any, expression string, currentPath []any) any {
	levels := 1

	expressionAndLocation := fmt.Sprintf("in expression: '%v'; at '%v'", expression, toPathExpression(currentPath))

	if len(args) == 2 {
		// Check if args[2] is an int
		v := args[1]
		levelArgs, ok := v.(int)
		if !ok {
			return fmt.Errorf("an int expected but got '%v': %v", v, expressionAndLocation)
		}
		levels = levelArgs
	}

	pathArg := args[0]
	path, ok := pathArg.([]any)
	if !ok {
		return fmt.Errorf("an array expected but got '%v': %v", pathArg, expressionAndLocation)
	}

	if len(path) < levels {
		return fmt.Errorf("value less than or equal to %v expected but got '%v': %v", len(path), levels, expressionAndLocation)
	}
	return path[0 : len(path)-levels]
}

// CreateRefFunc returns a builtin function descriptor for "ref".
// The builtin takes one argument: a path array. It returns the value at that
// path in self; if the value is a string, it is evaluated as an expression.
// baseDir and searchPaths are used when resolving file references in refs.
// The returned tuple is (name, minArgs, maxArgs, impl).
func CreateRefFunc(self any, currentPath []any, expression string, invocationSpec InvocationSpec, baseDir string, searchPaths []string, visited map[string]int) (string, int, int, func(any, []any) any) {
	return "ref", 1, 1, func(input any, args []any) any {
		pathArg := args[0]
		path, ok := pathArg.([]any)
		if !ok {
			return fmt.Errorf("expression: %s at %v; ret(%v); %v must be an array", expression, currentPath, args, pathArg)
		}

		return resolveRef(self, path, currentPath, invocationSpec, expression, baseDir, searchPaths, visited)
	}
}

// CreateRefExprFunc returns a builtin function descriptor for "refexpr".
// The builtin takes one argument: a path expression string (e.g. "$.foo.bar").
// It resolves the path and returns the value at that path; string values are
// evaluated as expressions. The returned tuple is (name, minArgs, maxArgs, impl).
func CreateRefExprFunc(self any, currentPath []any, expression string, invocationSpec InvocationSpec, visited map[string]int) (string, int, int, func(any, []any) any) {
	return "refexpr", 1, 1, func(input any, args []any) any {
		pathArg := args[0]
		pathexp, ok := pathArg.(string)
		if !ok {
			return fmt.Errorf("expression: %s at %v; ret(%v); %v must be a string", expression, currentPath, args, pathArg)
		}
		path, err := PathExpressionToPathArray(pathexp)
		if err != nil {
			return err
		}

		return resolveRef(self, path, currentPath, invocationSpec, expression, "", nil, visited)
	}
}

// CreateRefTagFunc returns a builtin function descriptor for "reftag".
// The builtin takes one argument: a tag name (string). It searches upward from
// the current path for an object with that tag and returns the value there;
// string values are evaluated as expressions. The returned tuple is (name, minArgs, maxArgs, impl).
func CreateRefTagFunc(self any, currentPath []any, expression string, invocationSpec InvocationSpec, visited map[string]int) (string, int, int, func(any, []any) any) {
	return "reftag", 1, 1, func(input any, args []any) any {
		tagArg := args[0]
		tag, ok := tagArg.(string)
		if !ok {
			return fmt.Errorf("expression: %s at %v; ret(%v); %v must be a string", expression, currentPath, args, tagArg)
		}
		pathLength := len(currentPath) - 1
		for i := pathLength; i >= 0; i-- {
			p := append(DeepCopy(currentPath[0:i]).([]any), tag)

			ret := resolveRef(self, p, currentPath, invocationSpec, expression, "", nil, visited)
			if _, ok := ret.(error); !ok {
				return ret
			}
		}

		return fmt.Errorf("tag: '%v' cannot be found from path: %v in obect: %v", tag, currentPath, marshal(self))
	}
}

// CreateReadFileFunc returns a builtin function descriptor for "readfile".
// The builtin takes one argument: a filename (string). The file is resolved
// relative to baseDir and searchPaths, read, and parsed as JSON. currentPath
// and expression are used for error reporting. The returned tuple is (name, minArgs, maxArgs, impl).
func CreateReadFileFunc(self any, currentPath []any, expression string, baseDir string, searchPaths []string) (string, int, int, func(any, []any) any) {
	return "readfile", 1, 1, func(input any, args []any) any {
		filenameArg := args[0]
		targetFile, ok := filenameArg.(string)
		if !ok {
			return fmt.Errorf("expression: %s at %v; ret(%v); %v must be an array", expression, currentPath, args, filenameArg)
		}
		absPath, err := ResolveFilePath(targetFile, baseDir, searchPaths)
		if err != nil {
			return err
		}
		ret, _, err := ReadFileAsJSONElement(absPath)
		if err != nil {
			return err
		}
		return ret
	}
}

// resolveRef looks up the value at path in self. If the value is a string, it
// is evaluated as an expression (e.g. a reference). visited is used to detect
// and reject circular references. Returns the value, or an error if the path
// is missing, already visited, or evaluation fails.
func resolveRef(self any, path []any, currentPath []any, invocationSpec InvocationSpec, expression string, baseDir string, searchPaths []string, visited map[string]int) any {
	pathexpr, err := PathArrayToPathExpression(path)
	if err != nil {
		panic(err)
	}
	if _, ok := visited[pathexpr]; ok {
		return fmt.Errorf("circular reference detected: %v", formatCirculatingFileLoop(extractVisitedLoop(visited, pathexpr), ""))
	}
	visited[pathexpr] = len(visited)
	if value, ok := GetAtPath(self, path); ok {
		// Process only if value is a string
		if str, ok := value.(string); ok {
			ret, err := evaluateString(str, currentPath, self, invocationSpec, baseDir, searchPaths, visited)
			if err != nil {
				return composeExpressionError(err, expression, currentPath)
			}
			delete(visited, pathexpr)
			return ret
		}
		delete(visited, pathexpr)
		return value
	}
	delete(visited, pathexpr)
	return fmt.Errorf("path: %v in expression: %v not found in object: %s", toPathExpression(path), expression, marshal(self))
}

func marshal(v any) any {
	ret, err := json.Marshal(v)
	if err != nil {
		return v
	}
	return string(ret)
}

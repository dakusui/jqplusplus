package internal

import (
	"fmt"
)

func CreateToPathArrayFunc() (string, int, int, func(any, []any) any) {
	return "topatharray", 1, 1, func(input any, args []any) any {
		ret, err := PathExpressionToPathArray(args[0].(string))
		if err != nil {
			return err
		}
		return ret
	}
}
func CreateToPathExprFunc() (string, int, int, func(any, []any) any) {
	return "topathexpr", 1, 1, func(input any, args []any) any {
		ret, err := PathArrayToPathExpression(args[0].([]any))
		if err != nil {
			return err
		}
		return ret
	}
}

func CreateParentOfFunc(currentPath []any, expression string) (string, int, int, func(any, []any) any) {
	return "parentof", 1, 2, func(input any, args []any) any {
		return resolveParent(args, expression, currentPath)
	}
}

func CreateParentFunc(currentPath []any, expression string) (string, int, int, func(any, []any) any) {
	return "parent", 0, 1, func(input any, args []any) any {
		return resolveParent(append([]any{currentPath}, args...), expression, currentPath)
	}
}

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

func CreateRefFunc(self any, currentPath []any, expression string, invocationSpec InvocationSpec, baseDir string, searchPaths []string) (string, int, int, func(any, []any) any) {
	visited := map[string]bool{}
	pathexpr, err := PathArrayToPathExpression(currentPath)
	if err != nil {
		panic(err)
	}
	visited[pathexpr] = true
	return "ref", 1, 1, func(input any, args []any) any {
		pathArg := args[0]
		path, ok := pathArg.([]any)
		if !ok {
			return fmt.Errorf("expression: %s at %v; ret(%v); %v must be an array", expression, currentPath, args, pathArg)
		}

		return resolveRef(self, path, currentPath, invocationSpec, expression, args, baseDir, searchPaths, visited)
	}
}

func CreateRefExprFunc(self any, currentPath []any, expression string, invocationSpec InvocationSpec) (string, int, int, func(any, []any) any) {
	visited := map[string]bool{}
	pathexpr, err := PathArrayToPathExpression(currentPath)
	if err != nil {
		panic(err)
	}
	visited[pathexpr] = true
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

		return resolveRef(self, path, currentPath, invocationSpec, expression, args, "", nil, visited)
	}
}

func CreateRefTagFunc(self any, currentPath []any, expression string, invocationSpec InvocationSpec) (string, int, int, func(any, []any) any) {
	visited := map[string]bool{}
	pathexpr, err := PathArrayToPathExpression(currentPath)
	if err != nil {
		panic(err)
	}
	visited[pathexpr] = true
	return "reftag", 1, 1, func(input any, args []any) any {
		tagArg := args[0]
		tag, ok := tagArg.(string)
		if !ok {
			return fmt.Errorf("expression: %s at %v; ret(%v); %v must be a string", expression, currentPath, args, tagArg)
		}
		pathLength := len(currentPath) - 1
		for i := pathLength; i >= 0; i-- {
			p := append(DeepCopy(currentPath[0:i]).([]any), tag)

			ret := resolveRef(self, p, currentPath, invocationSpec, expression, args, "", nil, visited)
			if _, ok := ret.(error); !ok {
				return ret
			}
		}

		return fmt.Errorf("tag: '%v' cannot be found from path: %v in obect: %v", tag, currentPath, self)
	}
}

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

func resolveRef(self any, path []any, currentPath []any, invocationSpec InvocationSpec, expression string, args []any, baseDir string, searchPaths []string, visited map[string]bool) any {
	pathexpr, err := PathArrayToPathExpression(path)
	if err != nil {
		panic(err)
	}
	if visited[pathexpr] {
		return fmt.Errorf("path expression: %v already visited: %v", pathexpr, currentPath)
	}
	visited[pathexpr] = true
	if value, ok := GetAtPath(self, path); ok {
		// Process only if value is a string
		if str, ok := value.(string); ok {
			ret, err := evaluateString(str, currentPath, self, invocationSpec, baseDir, searchPaths)
			if err != nil {
				return fmt.Errorf("expression: %s at %v; ref(%v) failed to eval for: %v", expression, path, args, err)
			}
			return ret
		}
		return value
	}
	return fmt.Errorf("path: %v in expression: %v not found in object: %v", path, expression, self)
}

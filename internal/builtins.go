package internal

import "fmt"

func createParentFunc(currentPath []any, expression string) (string, int, int, func(any, []any) any) {
	return "parent", 1, 2, func(input any, args []any) any {
		levels := 1
		if len(args) == 2 {
			// Check if args[2] is an int
			v := args[1]
			levelArgs, ok := v.(int)
			if !ok {
				return fmt.Errorf("expression: %s at %v; parent(%v); %v must be an int", expression, currentPath, args, v)
			}
			levels = levelArgs
		}

		pathArg := args[0]
		path, ok := pathArg.([]any)
		if !ok {
			return fmt.Errorf("expression: %s at %v; parent(%v); %v must be an array", expression, currentPath, args, pathArg)
		}

		if len(path) < levels {
			return fmt.Errorf("expression: %s at %v parent(%v); %v must be smaller than %v", expression, currentPath, args, levels, len(path))
		}
		return path[0 : len(path)-levels]
	}
}

func createRefFunc(self any, currentPath []any, expression string, invocationSpec InvocationSpec) (string, int, int, func(any, []any) any) {
	return "ref", 1, 1, func(input any, args []any) any {
		pathArg := args[0]
		path, ok := pathArg.([]any)
		if !ok {
			return fmt.Errorf("expression: %s at %v; ret(%v); %v must be an array", expression, currentPath, args, pathArg)
		}

		if value, ok := GetAtPath(self, path); ok {
			// Process only if value is a string
			if str, ok := value.(string); ok {
				ret, err := evaluateString(str, currentPath, self, invocationSpec)
				if err != nil {
					return fmt.Errorf("expression: %s at %v; ref(%v) failed to eval for: %v", expression, path, args, err)
				}
				return ret
			}
			return value
		}
		return fmt.Errorf("")
	}
}

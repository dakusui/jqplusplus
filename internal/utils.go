package internal

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func formatCirculatingFileLoop(visitedFiles []string, baseDir string) string {
	circulatingFiles := strings.Join(Map(visitedFiles,
		func(in string) string {
			if baseDir != "" && strings.HasPrefix(in, baseDir) {
				return strings.TrimPrefix(strings.TrimPrefix(in, baseDir), string(filepath.Separator))
			}
			return in
		}), " -> ")
	return circulatingFiles
}

func extractVisitedLoop(visited map[string]int, absPath string) []string {
	baseVal, ok := visited[absPath]
	if !ok {
		return nil
	}
	type pathVal struct {
		path string
		val  int
	}
	var rest []pathVal
	for path, val := range visited {
		if val >= baseVal {
			rest = append(rest, pathVal{path, val})
		}
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i].val < rest[j].val })
	result := make([]string, 0, 1+len(rest))
	for _, pv := range rest {
		result = append(result, pv.path)
	}
	return result
}

func composeExpressionError(err error, expression string, currentPath []any) error {
	if !strings.Contains(err.Error(), "expression: ") {
		return fmt.Errorf("%v: expression: %s at %v", err, expression, currentPath)
	}
	return err
}

func composeInheritanceResolutionErr(err error, level string) error {
	msg := fmt.Sprintf("failed to resolve %s-level inheritances", level)
	if strings.Contains(string(err.Error()), msg) {
		return err
	}
	return fmt.Errorf("%v: %s", err, msg)
}

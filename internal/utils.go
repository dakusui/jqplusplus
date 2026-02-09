package internal

import (
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
		if val > baseVal {
			rest = append(rest, pathVal{path, val})
		}
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i].val < rest[j].val })
	result := make([]string, 0, 2+len(rest))
	result = append(result, absPath)
	for _, pv := range rest {
		result = append(result, pv.path)
	}
	result = append(result, absPath)
	return result
}

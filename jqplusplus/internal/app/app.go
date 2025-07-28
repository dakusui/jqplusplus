package app

import (
	"os"
	"path/filepath"
	"strings"
)

type App struct {
	searchPaths []string
}

func (a *App) SearchPaths() []string {
	return NewApp().searchPaths
}

func (a *App) Run() {

}
func NewApp() *App {
	var paths []string
	pathsEnv := os.Getenv("JF_PATH")
	if pathsEnv != "" {
		paths = strings.Split(pathsEnv, ":")
	}
	return NewAppFromSearchPaths(paths)
}

func NewAppFromSearchPaths(searchPaths []string) *App {
	return &App{
		searchPaths: searchPaths,
	}
}

// ListSearchPaths generates the list of directories to search when resolving a file.
// This function works in the following way:
// 1. Resolve the directory of the current file
// 2. Add the directory of the current file to the search paths
// 3. Reads environment variable JF_PATH
// 4. Splits on :
// 5. Returns the list of directories
// 6. Prepend the search paths with the directories in JF_PATH
func (a *App) ListSearchPaths(currentFile string) []string {
	return append(a.SearchPaths(), filepath.Dir(currentFile))
}

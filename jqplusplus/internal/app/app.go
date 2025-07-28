package app

import (
	"os"
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

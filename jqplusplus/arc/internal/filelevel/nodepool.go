package filelevel

import (
	"path/filepath"
)

type NodePool interface {
	ReadNodeEntry(baseDir, filename string) (map[string]interface{}, error)
	IsVisited(absPath string) bool
	MarkVisited(absPath string)
	SearchPaths() []string
	SessionDirectory() string
	Enter(localNodeDirectory string)
	Leave(localNodeDirectory string)
}

type NodeEntry struct {
	filename string
	baseDir  string
}

func (n NodeEntry) String() string {
	return filepath.Join(n.baseDir, n.filename)
}

func (n NodeEntry) TemporaryDirectoryUnder(sessionDirectory string) string {
	return filepath.Join(sessionDirectory, n.String())
}

type StdNodePool struct {
	baseDir              string
	sessionDirectory     string
	localNodeSearchPaths []string
	baseSearchPaths      []string
	cache                map[NodeEntry]map[string]any
	visited              map[string]bool
}

func NewStdNodePoolWithBaseSearchPaths(baseDir, sessionDirectory string, searchPaths []string) *StdNodePool {
	return &StdNodePool{
		baseDir:              baseDir,
		sessionDirectory:     sessionDirectory,
		localNodeSearchPaths: []string{},
		baseSearchPaths:      searchPaths,
		cache:                map[NodeEntry]map[string]any{},
		visited:              map[string]bool{},
	}
}

func (p *StdNodePool) ReadNodeEntry(baseDir, filename string) (obj map[string]interface{}, err error) {
	nodeEntry := NodeEntry{filename: filename, baseDir: baseDir}
	ret, ok := p.cache[nodeEntry]
	if !ok {
		ret, err = LoadAndResolveInheritancesRecursively(baseDir, filename, p)
		if err != nil {
			return nil, err
		}
		p.cache[nodeEntry] = ret
	}
	return ret, nil
}

func (p *StdNodePool) IsVisited(absPath string) bool {
	return p.visited[absPath]
}

func (p *StdNodePool) MarkVisited(absPath string) {
	p.visited[absPath] = true
}

func (p *StdNodePool) Enter(localNodeDirectory string) {
	p.localNodeSearchPaths = append(p.localNodeSearchPaths, localNodeDirectory)
}

func (p *StdNodePool) Leave(localNodeDirectory string) {
	if localNodeDirectory != p.localNodeSearchPaths[len(p.localNodeSearchPaths)-1] {
		panic("Unexpected leave")
	}
	p.localNodeSearchPaths = p.localNodeSearchPaths[:len(p.localNodeSearchPaths)-1]
}

func (p *StdNodePool) SessionDirectory() string {
	return p.sessionDirectory
}

func (p *StdNodePool) SearchPaths() []string {
	paths := make([]string, 0, 1+len(p.localNodeSearchPaths)+len(p.baseSearchPaths))

	if p.baseDir != "" {
		paths = append(paths, p.baseDir)
	}
	paths = append(paths, p.localNodeSearchPaths...)
	paths = append(paths, p.baseSearchPaths...)

	return Filter(paths, func(p string) bool { return p != "" })
}

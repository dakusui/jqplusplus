package filelevel

type NodePool interface {
	ReadNodeEntry(baseDir, filename string) (map[string]interface{}, error)
	IsVisited(absPath string) bool
	MarkVisited(absPath string)
	SearchPaths() []string
}

type StdNodePool struct {
	baseDir              string
	localNodeSearchPaths []string
	baseSearchPaths      []string
	cache                map[NodeEntry]map[string]any
	visited              map[string]bool
}

type NodeEntry struct {
	filename string
	baseDir  string
}

func NewStdNodePoolWithSearchPaths(baseDir string, searchPaths []string) *StdNodePool {
	return &StdNodePool{
		baseDir:              baseDir,
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

func (p *StdNodePool) SearchPaths() []string {
	paths := make([]string, 0, 1+len(p.localNodeSearchPaths)+len(p.baseSearchPaths))

	if p.baseDir != "" {
		paths = append(paths, p.baseDir)
	}
	paths = append(paths, p.localNodeSearchPaths...)
	paths = append(paths, p.baseSearchPaths...)

	return paths
}

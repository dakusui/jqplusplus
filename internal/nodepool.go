package internal

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

type NodePoolImpl struct {
	baseDir              string
	sessionDirectory     string
	localNodeSearchPaths []string
	baseSearchPaths      []string
	cache                map[NodeEntry]map[string]any
	visited              map[string]bool
}

func NewNodePoolWithBaseSearchPaths(baseDir, sessionDirectory string, searchPaths []string) *NodePoolImpl {
	return &NodePoolImpl{
		baseDir:              baseDir,
		sessionDirectory:     sessionDirectory,
		localNodeSearchPaths: []string{},
		baseSearchPaths:      searchPaths,
		cache:                map[NodeEntry]map[string]any{},
		visited:              map[string]bool{},
	}
}

func (p *NodePoolImpl) ReadNodeEntry(baseDir, filename string) (obj map[string]interface{}, err error) {
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

func (p *NodePoolImpl) IsVisited(absPath string) bool {
	return p.visited[absPath]
}

func (p *NodePoolImpl) MarkVisited(absPath string) {
	p.visited[absPath] = true
}

func (p *NodePoolImpl) Enter(localNodeDirectory string) {
	p.localNodeSearchPaths = append(p.localNodeSearchPaths, localNodeDirectory)
}

func (p *NodePoolImpl) Leave(localNodeDirectory string) {
	if localNodeDirectory != p.localNodeSearchPaths[len(p.localNodeSearchPaths)-1] {
		panic("Unexpected leave")
	}
	p.localNodeSearchPaths = p.localNodeSearchPaths[:len(p.localNodeSearchPaths)-1]
}

func (p *NodePoolImpl) SessionDirectory() string {
	return p.sessionDirectory
}

func (p *NodePoolImpl) SearchPaths() []string {
	paths := make([]string, 0, 1+len(p.localNodeSearchPaths)+len(p.baseSearchPaths))

	if p.baseDir != "" {
		paths = append(paths, p.baseDir)
	}
	paths = append(paths, p.localNodeSearchPaths...)
	paths = append(paths, p.baseSearchPaths...)

	return Filter(paths, func(p string) bool { return p != "" })
}

package filelevel

type NodePool interface {
	ReadNodeEntry(baseDir, filename string) (map[string]interface{}, error)
	IsVisited(absPath string) bool
	MarkVisited(absPath string)
	SearchPaths() []string
}

type StdNodePool struct {
	searchPaths []string
	cache       map[string]map[string]map[string]interface{}
	visited     map[string]bool
}

func NewStdNodePoolWithSearchPaths(searchPaths []string) *StdNodePool {
	return &StdNodePool{
		searchPaths: searchPaths,
		cache:       map[string]map[string]map[string]interface{}{},
		visited:     map[string]bool{},
	}
}

func (p *StdNodePool) ReadNodeEntry(baseDir, filename string) (obj map[string]interface{}, err error) {
	files, ok := p.cache[baseDir]
	if !ok {
		ret, err := LoadAndResolveInheritancesRecursively(baseDir, filename, p)
		if err != nil {
			return nil, err
		}
		p.cache[baseDir] = map[string]map[string]interface{}{}
		p.cache[baseDir][filename] = ret
		return ret, nil
	}
	ret, ok := files[filename]
	if !ok {
		ret, err := LoadAndResolveInheritancesRecursively(baseDir, filename, p)
		if err != nil {
			return nil, err
		}
		files[filename] = ret
		return ret, nil
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
	return p.searchPaths
}

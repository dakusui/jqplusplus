package utils

type NodePool struct {
	cache  map[string]map[string]interface{}
	loader NodeLoader
}

type NodeLoader interface {
	LoadNode(n NodeUnit) (map[string]interface{}, error)
}

type simpleNodeLoader struct {
	paths []string
}

func NewSimpleNodeLoader(paths []string) *simpleNodeLoader {
	return &simpleNodeLoader{paths: paths}
}

func (l *simpleNodeLoader) LoadNode(n NodeUnit) (map[string]interface{}, error) {
	f, err := FindFileInPath(n.name, l.paths)
	if err != nil {
		return nil, err
	}
	decoder, err := CreateDecoder(n.decoder, n.args)
	if err != nil {
		return nil, err
	}
	ret, err := ReadFileAsJSONObject(f, decoder)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func NewNodePool(loader NodeLoader) *NodePool {
	return &NodePool{
		cache:  map[string]map[string]interface{}{},
		loader: loader,
	}
}

func NewNodePoolWithSimpleLoader(paths []string) *NodePool {
	return NewNodePool(NewSimpleNodeLoader(paths))
}

func (p *NodePool) GetNode(n NodeUnit) (map[string]interface{}, error) {
	value, ok := p.cache[n.String()]
	if !ok {
		v, err := p.loader.LoadNode(n)
		if err != nil {
			return nil, err
		}
		p.cache[n.String()] = v
		value = v
	}
	return value, nil
}

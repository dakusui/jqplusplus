package utils

type NodePool struct {
	cache  map[string]map[string]interface{}
	loader NodeLoader
}

type NodeLoader interface {
	LoadNode(n NodeUnit) (map[string]interface{}, error)
}

func NewNodePool(loader NodeLoader) *NodePool {
	return &NodePool{
		cache:  map[string]map[string]interface{}{},
		loader: loader,
	}
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

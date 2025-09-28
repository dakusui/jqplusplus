package utils

type NodePool struct {
	cache  map[string]map[string]interface{}
	loader NodeLoader
}

type NodeLoader interface {
	// LoadNode loads a NodeUnit based on the provided identifier and returns its content as a map or an error if it fails.
	// `by` specifies a file by which `n` is read.
	// `n` will be resolved
	LoadNode(n NodeUnit) (map[string]interface{}, error)
}

func (p *NodePool) GetNode(n NodeUnit, by string) (map[string]interface{}, error) {
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

func NewNodePool(loader NodeLoader) *NodePool {
	return &NodePool{
		cache:  map[string]map[string]interface{}{},
		loader: loader,
	}
}

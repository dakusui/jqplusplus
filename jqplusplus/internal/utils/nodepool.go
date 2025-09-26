package utils

import (
	"path/filepath"
)

type NodePool struct {
	cache  map[string]map[string]interface{}
	loader NodeLoader
}

type NodeLoader interface {
	LoadNode(n NodeUnit, by string) (map[string]interface{}, error)
}

type simpleNodeLoader struct {
	paths []string
}

func (l simpleNodeLoader) LoadNode(n NodeUnit, by string) (map[string]interface{}, error) {
	parent := filepath.Dir(by)
	f, err := FindFileInPath(n.name, Insert(l.paths, 0, parent))
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

func NewSimpleNodeLoader(paths []string) NodeLoader {
	ret := &simpleNodeLoader{paths: paths}
	return ret
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

func (p *NodePool) GetNode(n NodeUnit, by string) (map[string]interface{}, error) {
	value, ok := p.cache[n.String()]
	if !ok {
		v, err := p.loader.LoadNode(n, by)
		if err != nil {
			return nil, err
		}
		p.cache[n.String()] = v
		value = v
	}
	return value, nil
}

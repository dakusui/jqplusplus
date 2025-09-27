package utils

import "path/filepath"

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

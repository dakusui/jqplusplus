package filelevel

type NodePool interface {
	ReadNodeEntry(nodeEntry NodeEntry) map[string]interface{}
}

type NodeEntry struct {
	BaseDir  string
	FileName string
}

type StdNodePool struct {
	searchPaths []string
	cache       map[string]map[string]interface{}
}

func (f *StdNodePool) ReadNodeEntry(nodeEntry NodeEntry) map[string]interface{} {
	return nil
}

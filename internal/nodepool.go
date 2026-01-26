package internal

import (
	"path/filepath"
)

type NodePool interface {
	ReadNodeEntryValue(baseDir, filename string, compilerOptions []*JqModule) (*NodeEntryValue, error)
	IsVisited(absPath string) bool
	MarkVisited(absPath string)
	SearchPaths() []string
	SessionDirectory() string

	// Enter adds a localNodeDirectory to the localNodeSearchPaths stack
	// of the NodePool. This method can be used to manage the context
	// of a node, where nodes are processed in a nested structure.
	// If "" is passed as localNodeDirectory, this method will do nothing.
	//
	// Parameters:
	// - localNodeDirectory: The directory path to be added to the search paths.
	//
	// Output:
	// - This function outputs the added localNodeDirectory to stderr for debugging purposes.
	//
	// Example usage:
	//   pool.Enter("/path/to/localNodeDir")
	//
	// Note: Ensure that for every call to Enter, a corresponding call
	// to Leave is made to maintain the correct stack structure.
	Enter(localNodeDirectory string)

	// Leave removes the specified localNodeDirectory from the localNodeSearchPaths stack of the NodePool.
	// If "" is passed as localNodeDirectory, this method will do nothing.
	Leave(localNodeDirectory string)
}

type NodePoolImpl struct {
	baseDir              string
	sessionDirectory     string
	localNodeSearchPaths []string
	// Paths from which files to be inherited are searched for.
	baseSearchPaths []string
	// cache holds the mapping of NodeEntryKey to NodeEntryValue, providing
	// a mechanism for caching node entries that have been processed. This
	// ensures that previously resolved entries can be retrieved efficiently
	// without redundant operations.
	cache   map[NodeEntryKey]NodeEntryValue
	visited map[string]bool
}

func NewNodePoolWithBaseSearchPaths(baseDir, sessionDirectory string, searchPaths []string) *NodePoolImpl {
	return &NodePoolImpl{
		baseDir:              baseDir,
		sessionDirectory:     sessionDirectory,
		localNodeSearchPaths: []string{},
		baseSearchPaths:      searchPaths,
		cache:                map[NodeEntryKey]NodeEntryValue{},
		visited:              map[string]bool{},
	}
}

func (p *NodePoolImpl) ReadNodeEntryValue(baseDir, filename string, compilerOptions []*JqModule) (*NodeEntryValue, error) {
	nodeEntryKey := NodeEntryKey{filename: filename, baseDir: baseDir}
	ret, ok := p.cache[nodeEntryKey]
	if !ok {
		nodeEntryValue, err := LoadAndResolveInheritancesRecursively(baseDir, filename, p)
		if err != nil {
			return nil, err
		}
		p.cache[nodeEntryKey] = *nodeEntryValue
		ret = *nodeEntryValue
	}
	ret.CompilerOptions = append(compilerOptions, ret.CompilerOptions...)
	return &ret, nil
}

func (p *NodePoolImpl) IsVisited(absPath string) bool {
	return p.visited[absPath]
}

func (p *NodePoolImpl) MarkVisited(absPath string) {
	p.visited[absPath] = true
}

func (p *NodePoolImpl) Enter(localNodeDirectory string) {
	if localNodeDirectory == "" {
		return
	}
	p.localNodeSearchPaths = append(p.localNodeSearchPaths, localNodeDirectory)
}

func (p *NodePoolImpl) Leave(localNodeDirectory string) {
	if localNodeDirectory == "" {
		return
	}
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

type NodeEntryKey struct {
	filename string
	baseDir  string
}

func NewNodeEntryKey(baseDir, filename string) NodeEntryKey {
	return NodeEntryKey{filename: filename, baseDir: baseDir}
}

// NodeEntryValue represents the value corresponding to a NodeEntryKey in the NodePool cache.
// It encapsulates a map of objects and a list of gojq.CompilerOption used for processing
// jq queries.
//
// Fields:
// - Obj: A map containing arbitrary data associated with the NodeEntry.
// - CompilerOptions: A list of options applied when compiling jq queries.
type NodeEntryValue struct {
	Obj             map[string]any
	CompilerOptions []*JqModule
}

func (e NodeEntryKey) BaseDir() string {
	return e.baseDir
}

func (e NodeEntryKey) Filename() string {
	return e.filename
}

func (e NodeEntryKey) String() string {
	return filepath.Join(e.BaseDir(), e.Filename())
}

func NewNodeEntry(baseDir, filename string) NodeEntryKey {
	return NodeEntryKey{filename: filename, baseDir: baseDir}
}

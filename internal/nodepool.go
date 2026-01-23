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
	Enter(localNodeDirectory string)
	Leave(localNodeDirectory string)
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

package bandit

type (
	Tree struct {
		nodes *Nodes
		root  NodeID
		ul    bool
	}
)

func NewTree(nodes *Nodes) Tree {
	return Tree{
		nodes: nodes,
		root:  nodes.Alloc(),
	}
}

func (t *Tree) Free() {
	t.nodes.FreeAll(t.root)
}

package bandit

type (
	Tree struct {
		root NodeID
		ul   bool
	}
)

func NewTree(nodes *Nodes) Tree {
	return Tree{
		root: nodes.Alloc(),
	}
}

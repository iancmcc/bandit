package bandit

type (
	NodeID int32
	Node   struct {
		Prefix uint64
		Level  int32
		Parent NodeID
		Left   NodeID
		Right  NodeID
		Count  int32
		Ul     bool
		Incl   bool
	}
	Nodes struct {
		bucket_size int
		current     NodeID
		next_free   NodeID
		num_free    int
		remaining   int
		storage     [][]Node
	}
)

func NewNodes(bucket_size int) *Nodes {
	return &Nodes{
		bucket_size: bucket_size,
		storage:     make([][]Node, 0),
	}
}

func (n *Nodes) Get(id NodeID) *Node {
	i := int(id) - 1
	bucket := i / n.bucket_size
	idx := i % n.bucket_size
	return &n.storage[bucket][idx]
}

func (n *Nodes) Alloc() NodeID {
	if n.num_free > 0 {
		idx := n.next_free
		n.next_free = n.Get(idx).Left
		if n.next_free == 0 {
			n.num_free = 0
		} else {
			n.num_free -= 1
		}
		return idx
	}
	if n.remaining == 0 && n.num_free == 0 {
		new_bucket := make([]Node, n.bucket_size, n.bucket_size)
		n.storage = append(n.storage, new_bucket)
		n.remaining = n.bucket_size
	}
	n.remaining -= 1
	n.current += 1
	return n.current
}

func (n *Nodes) Free(id NodeID) {
	*(n.Get(id)) = Node{Left: n.next_free}
	n.next_free = id
	n.num_free += 1
}

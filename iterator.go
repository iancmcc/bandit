package bandit

type IntervalIterator struct {
	t         Tree
	nodestack []uint
	ulstack   []bool
}

func NewIntervalIterator(t Tree) *IntervalIterator {
	numnodes := len(t.nodes) - int(t.numfree)
	nodestack := append(make([]uint, 0, numnodes), t.root)
	ulstack := append(make([]bool, 0, numnodes), t.ul)
	return &IntervalIterator{
		t:         t,
		nodestack: nodestack,
		ulstack:   ulstack,
	}
}

func (it *IntervalIterator) Next() (ival Interval) {
	var (
		l         = len(it.nodestack)
		n         uint
		ul        bool
		cur       node
		idx, left uint
	)
	for ; l > 0; l = len(it.nodestack) {
		n, it.nodestack = it.nodestack[l-1], it.nodestack[:l-1]
		ul, it.ulstack = it.ulstack[l-1], it.ulstack[:l-1]
		cur = it.t.nodes[n]
		if cur.level != 0 {
			// This is an internal node, so add its children (right first, so
			// we visit left first)
			it.nodestack = append(it.nodestack, cur.right, cur.left)
			// Reuse cur for left
			cur = it.t.nodes[cur.left]
			it.ulstack = append(it.ulstack, ul != cur.ul, ul)
			continue
		}
		// cur is a leaf
		if cur.ul {
			// This node is part of a single interval
			idx = ival.Tree.takeOwnership(&it.t, n)
			if !ul {
				// Opening an interval
				left = idx
				continue
			}
			// Closing an interval
			ival.mergeRoot(&ival.Tree, &ival.Tree, left, idx, true, true, and)
		} else {
			// This node is either a hole or a point
			if !ul {
				// Point
				ival.root = ival.Tree.takeOwnership(&it.t, n)
				return
			}
			// Hole
			// Close what we've got
			ival.mergeRoot(&ival.Tree, &ival.Tree, left, idx, true, true, and)
			// Push this node back on the stack, updating it to open the next
			// interval
			cur.ul = true
			it.nodestack = append(it.nodestack, n)
			it.ulstack = append(it.ulstack, false)
		}
		return
	}
	ival.mergeRoot(&ival.Tree, &ival.Tree, left, 0, true, true, and)
	return
}

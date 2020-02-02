package bandit

type (
	IntervalIterator struct {
		t         Tree
		nodestack []uint
		ulstack   []bool
		ival      Interval
		done      bool
		holeleft  bool
	}
	MapIterator struct {
		m         *IntervalMap
		this      interface{}
		remaining []interface{}
	}
)

func NewMapIterator(m *IntervalMap) *MapIterator {
	remaining := make([]interface{}, 0, len(m.m))
	for k := range m.m {
		remaining = append(remaining, k)
	}
	return &MapIterator{m: m, remaining: remaining}
}

func (mi *MapIterator) Value() (interface{}, *IntervalSet) {
	return mi.this, &mi.m.sets[mi.m.m[mi.this]].IntervalSet
}

func (mi *MapIterator) Next() bool {
	if len(mi.remaining) == 0 {
		return false
	}
	mi.this, mi.remaining = mi.remaining[0], mi.remaining[1:]
	return true
}

func NewIntervalIterator(t Tree) *IntervalIterator {
	numnodes := len(t.nodes) - int(t.numfree)
	nodestack := append(make([]uint, 0, numnodes), t.root)
	ulstack := append(make([]bool, 0, numnodes), t.ul)
	return &IntervalIterator{
		t:         t,
		nodestack: nodestack,
		ulstack:   ulstack,
		ival:      Empty(),
	}
}

func (it *IntervalIterator) Interval() (ival Interval) {
	return it.ival
}
func (it *IntervalIterator) Next() bool {
	if it.done {
		return false
	}
	var (
		l         = len(it.nodestack)
		n         uint
		ul        bool
		lul       bool = it.t.ul
		cur       node
		idx, left uint
	)
	if l == 0 || it.nodestack[0] == 0 {
		it.done = true
		return false
	}
	it.ival.Clear()
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
		if it.holeleft || cur.ul {
			// This node is part of a single interval
			idx = it.ival.Tree.takeOwnership(&it.t, n)
			if !ul {
				if it.holeleft {
					(&it.ival.nodes[idx]).incl = false
					it.holeleft = false
					ul = !ul
				}
				// Opening an interval
				left = idx
				lul = ul
				continue
			}
			// Closing an interval
			it.ival.mergeRoot(&it.ival.Tree, &it.ival.Tree, left, idx, lul, ul, and)
		} else {
			// This node is either a hole or a point
			if !ul {
				// Point
				it.ival.root = it.ival.Tree.takeOwnership(&it.t, n)
				// (&it.ival.nodes[it.ival.root]).incl = true // TODO: Make a test for point
				return true
			}
			idx = it.ival.Tree.takeOwnership(&it.t, n)
			// Hole
			// Close what we've got
			it.ival.mergeRoot(&it.ival.Tree, &it.ival.Tree, left, idx, lul, ul, and)

			// Push this node back on the stack, setting a flag so it will be
			// treated only as the left side of an interval
			it.holeleft = true
			it.nodestack = append(it.nodestack, n)
			it.ulstack = append(it.ulstack, false)
		}
		return true
	}
	if left > 0 {
		it.ival.mergeRoot(&it.ival.Tree, &it.ival.Tree, left, 0, false, true, and)
	}
	if it.ival.IsEmpty() {
		it.done = true
		return false
	}
	return true
}

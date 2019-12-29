package bandit

import (
	"strings"
)

type (
	IntervalSet struct {
		Tree
	}
)

const defaultNodeArraySize = 32

func NewIntervalSetWithCapacity(capacity uint, intervals ...Interval) *IntervalSet {
	var set IntervalSet
	set.nodes = make([]node, 1, capacity)
	for _, ival := range intervals {
		set.mergeRoot(&set.Tree, &ival.Tree, set.root, ival.root, set.ul, ival.ul, or)
	}
	return &set
}

func NewIntervalSet(intervals ...Interval) *IntervalSet {
	return NewIntervalSetWithCapacity(defaultNodeArraySize, intervals...)
}

func (z *IntervalSet) Iterator() *IntervalIterator {
	return NewIntervalIterator(z.Tree)
}

func (z *IntervalSet) String() string {
	s := []string{}
	iterator := z.Iterator()
	for iterator.Next() {
		s = append(s, iterator.Interval().String())
	}
	return strings.Join(s, ", ")
}

func (z *IntervalSet) Add(ival ...Interval) *IntervalSet {
	for _, iv := range ival {
		z.mergeRoot(&z.Tree, &iv.Tree, z.root, iv.root, z.ul, iv.ul, or)
	}
	return z
}

func (z *IntervalSet) Complement(x *IntervalSet) *IntervalSet {
	if z != x {
		z.Clear()
		z.Union(z, x)
	}
	z.ul = !z.ul
	return z
}

func (z *IntervalSet) CommonIntervals(x, y *IntervalSet) *IntervalSet {
	if z != x && z != y {
		z.Clear()
	}
	z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, y.ul, common)
	return z
}

func (z *IntervalSet) Enclosing(x, y *IntervalSet) *IntervalSet {
	if z != x && z != y {
		z.Clear()
	}
	z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, y.ul, and)
	z.mergeRoot(&z.Tree, &y.Tree, z.root, y.root, z.ul, y.ul, common)
	return z
}

func (z *IntervalSet) Intersection(x, y *IntervalSet) *IntervalSet {
	if z != x && z != y {
		z.Clear()
	}
	z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, y.ul, and)
	return z
}

func (z *IntervalSet) Union(x, y *IntervalSet) *IntervalSet {
	if z != x && z != y {
		z.Clear()
	}
	z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, y.ul, or)
	return z
}

func (z *IntervalSet) SymmetricDifference(x, y *IntervalSet) *IntervalSet {
	if z != x && z != y {
		z.Clear()
	}
	z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, y.ul, xor)
	return z
}

func (z *IntervalSet) Difference(x, y *IntervalSet) *IntervalSet {
	if z != x && z != y {
		z.Clear()
	}
	z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, !y.ul, and)
	return z
}

func (z *IntervalSet) Equals(other *IntervalSet) bool {
	return z.ul == other.ul && treeEquals(&z.Tree, &other.Tree, z.root, other.root)
}

func (z *IntervalSet) IsEmpty() bool {
	return z.root == 0 && !z.ul
}

func (z *IntervalSet) IntervalContaining(val uint64) (ival Interval) {
	idx, ul := z.Tree.leftEdge(val)
	n := &z.Tree.nodes[idx]
	switch {
	case idx == 0:
		if !z.ul {
			return Empty()
		}
		ival = Unbounded()
	case ul && (n.prefix == val || !n.boundBoth()):
		return Empty()
	case n.boundAbove(), n.boundBoth():
		ival = Above(n.prefix)
	case n.boundBelow():
		ival = AtOrAbove(n.prefix)
	}
	idx, ul = z.Tree.rightEdge(val)
	if idx == 0 {
		return
	} else if !ul {
		return Empty()
	}
	n = &z.Tree.nodes[idx]

	var rival Interval
	switch {
	case n.boundBelow(), n.boundBoth():
		rival = Below(n.prefix)
	case n.boundAbove():
		rival = AtOrBelow(n.prefix)
	}
	ival = ival.Intersection(rival)
	return
}

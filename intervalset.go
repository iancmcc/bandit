package bandit

import (
	"strings"
)

type (
	IntervalSet struct {
		Tree
	}
)

const defaultIntervalSetCapacity = 8

func NewSet(capacity int) *IntervalSet {
	return NewIntervalSetWithCapacity(uint(capacity))
}

func NewIntervalSetWithCapacity(capacity uint, intervals ...Interval) *IntervalSet {
	if capacity == 0 {
		capacity = 1
	}
	set := newIntervalSetWithNodeStorage(make([]node, 1, capacity*5+1), intervals...)
	return &set
}

func NewIntervalSet(intervals ...Interval) *IntervalSet {
	return NewIntervalSetWithCapacity(defaultIntervalSetCapacity, intervals...)
}

func newIntervalSetWithNodeStorage(storage []node, intervals ...Interval) IntervalSet {
	var set IntervalSet
	set.nodes = storage[:1]
	for _, ival := range intervals {
		set.mergeRoot(&set.Tree, &ival.Tree, set.root, ival.root, set.ul, ival.ul, or)
	}
	return set
}

func (z *IntervalSet) Cap() int {
	return cap(z.nodes)
}

func (z *IntervalSet) Iterator() *IntervalIterator {
	return NewIntervalIterator(z.Tree)
}

func (z *IntervalSet) String() string {
	if z.root == 0 {
		if z.ul {
			return infinite
		}
		return empty
	}
	s := make([]string, 0, z.Cardinality())
	for iterator := z.Iterator(); iterator.Next(); {
		s = append(s, iterator.Interval().String())
	}
	return strings.Join(s, ", ")
}

func (z *IntervalSet) Copy(x *IntervalSet) *IntervalSet {
	switch {
	case z == x:
		return x
	case x == nil:
		z.Clear()
	default:
		z.root = x.root
		z.nextfree = x.nextfree
		z.numfree = x.numfree
		z.ul = x.ul
		z.nodes = append(z.nodes[:0:0], x.nodes...)
	}
	return z
}

func (z *IntervalSet) Add(x *IntervalSet, ival ...Interval) *IntervalSet {
	if z != x {
		z.Clear()
		z.Copy(x)
	}
	for _, iv := range ival {
		z.mergeRoot(&z.Tree, &iv.Tree, z.root, iv.root, z.ul, iv.ul, or)
	}
	return z
}

func (z *IntervalSet) Complement(x *IntervalSet) *IntervalSet {
	switch {
	case x == nil:
		z.Clear()
	case z != x:
		z.Copy(x)
	}
	z.ul = !z.ul
	return z
}

func (z *IntervalSet) Cardinality() int {
	if z.root == 0 {
		if z.ul {
			return 1
		}
		return 0
	}
	i := (&z.nodes[z.root]).count / 2
	if z.ul || i == 0 {
		i += 1
	}
	return int(i)
}

func (z *IntervalSet) FirstInterval() Interval {
	if z.IsEmpty() {
		return Empty()
	}
	if z.IsUnbounded() {
		return Unbounded()
	}
	var ival Interval
	l, ul := z.leftmostLeaf(z.root, z.ul)
	if ul {
		ival.Tree.mergeRoot(&z.Tree, &z.Tree, 0, l, true, z.ul, and)
	} else {
		ival.Tree.mergeRoot(&z.Tree, &z.Tree, l, z.nextLeaf(l), ul, !ul, and)
	}
	return ival
}

func (z *IntervalSet) Extent() Interval {
	if z.IsEmpty() {
		return Empty()
	}
	if z.IsUnbounded() {
		return Unbounded()
	}
	ival := Empty()
	//z.Check()
	l, lul := z.leftmostLeaf(z.root, z.ul)
	r, rul := z.rightmostLeaf(z.root, z.ul)
	ival.Tree.mergeRoot(&z.Tree, &z.Tree, l, r, lul, rul, and)
	//ival.Check()
	return ival
}

func (z *IntervalSet) Intersection(x, y *IntervalSet) *IntervalSet {
	switch {
	case x == nil, y == nil:
		z.Clear()
	case z != x && z != y:
		z.Clear()
		fallthrough
	default:
		z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, y.ul, and)
		//z.Check()
	}
	return z
}

func (z *IntervalSet) Union(x, y *IntervalSet) *IntervalSet {
	switch {
	case x == nil || x.IsEmpty():
		z.Copy(y)
	case y == nil || y.IsEmpty():
		z.Copy(x)
	case z != x && z != y:
		z.Clear()
		fallthrough
	default:
		/*
			z.Check()
			x.Check()
			y.Check()
		*/
		z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, y.ul, or)
	}
	return z
}

func (z *IntervalSet) SymmetricDifference(x, y *IntervalSet) *IntervalSet {
	switch {
	case x == nil:
		z.Copy(y)
	case y == nil:
		z.Copy(x)
	case z != x && z != y:
		z.Clear()
		fallthrough
	default:
		z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, y.ul, xor)
	}
	return z
}

func (z *IntervalSet) Difference(x, y *IntervalSet) *IntervalSet {
	switch {
	case x == nil:
		z.Clear()
	case y == nil:
		z.Copy(x)
	case z != x && z != y:
		z.Clear()
		fallthrough
	default:
		z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, !y.ul, and)
	}
	return z
}

func (z *IntervalSet) Equals(other *IntervalSet) bool {
	if other == nil {
		return z.IsEmpty()
	}
	return z.ul == other.ul && treeEquals(&z.Tree, &other.Tree, z.root, other.root)
}

func (z *IntervalSet) IsEmpty() bool {
	return z.root == 0 && !z.ul
}

func (z *IntervalSet) IsUnbounded() bool {
	return z.root == 0 && z.ul
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

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

func NewIntervalSetWithCapacity(capacity uint, intervals ...Interval) *IntervalSet {
	set := newIntervalSetWithNodeStorage(make([]node, 1, capacity*4), intervals...)
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

func (z *IntervalSet) CommonIntervals(x, y *IntervalSet) *IntervalSet {
	switch {
	case x.ul == true, y.ul == true:
		panic("Finding common left-unbounded intervals isn't supported")
	case x == nil, y == nil:
		z.Clear()
	case z != x && z != y:
		z.Clear()
		fallthrough
	default:
		z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, y.ul, common)
	}
	return z
}

func (z *IntervalSet) Enclosed(x, y *IntervalSet) *IntervalSet {
	switch {
	case x.ul == true, y.ul == true:
		panic("Finding enclosed/enclosing left-unbounded intervals isn't supported")
	case x == nil, y == nil:
		z.Clear()
	case z != x && z != y:
		z.Clear()
		fallthrough
	default:
		z.mergeRoot(&x.Tree, &y.Tree, x.root, y.root, x.ul, y.ul, and)
		z.mergeRoot(&z.Tree, &y.Tree, z.root, y.root, z.ul, y.ul, common)
	}
	return z
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
	}
	return z
}

func (z *IntervalSet) Union(x, y *IntervalSet) *IntervalSet {
	switch {
	case x == nil:
		z.Copy(y)
	case y == nil:
		z.Copy(x)
	case z != x && z != y:
		z.Clear()
		fallthrough
	default:
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

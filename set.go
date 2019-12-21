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

func (z *IntervalSet) Add(ival Interval) *IntervalSet {
	z.mergeRoot(&z.Tree, &ival.Tree, z.root, ival.root, z.ul, ival.ul, or)
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

func (z *IntervalSet) Intersection(x, y *IntervalSet) *IntervalSet {
	switch {
	case z == x:
	case z == y:
		x, y = y, x
	default:
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

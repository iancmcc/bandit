package bandit

import "strings"

type (
	IntervalSet struct {
		Tree
		array [1024]node
	}
)

func NewIntervalSet(intervals ...Interval) (set IntervalSet) {
	set.nodes = set.array[:1]
	for _, ival := range intervals {
		set.mergeRoot(&set.Tree, &ival.Tree, set.root, ival.root, set.ul, ival.ul, or)
	}
	return
}

func (set IntervalSet) Iterator() *IntervalIterator {
	return NewIntervalIterator(set.Tree)
}

func (set IntervalSet) String() string {
	s := []string{}
	iterator := set.Iterator()
	for iterator.Next() {
		s = append(s, iterator.Interval().String())
	}
	return strings.Join(s, ", ")
}

func (set IntervalSet) Add(ival Interval) IntervalSet {
	set.nodes = set.nodes[:len(set.nodes)]
	set.mergeRoot(&set.Tree, &ival.Tree, set.root, ival.root, set.ul, ival.ul, or)
	return set
}

func (set IntervalSet) Complement() IntervalSet {
	set.nodes = set.nodes[:len(set.nodes)]
	set.ul = !set.ul
	return set
}

func (set IntervalSet) Intersection(other IntervalSet) IntervalSet {
	set.nodes = set.nodes[:len(set.nodes)]
	set.mergeRoot(&set.Tree, &other.Tree, set.root, other.root, set.ul, other.ul, and)
	return set
}

func (set IntervalSet) Union(other IntervalSet) IntervalSet {
	set.nodes = set.nodes[:len(set.nodes)]
	set.mergeRoot(&set.Tree, &other.Tree, set.root, other.root, set.ul, other.ul, or)
	return set
}

func (set IntervalSet) SymmetricDifference(other IntervalSet) IntervalSet {
	set.nodes = set.nodes[:len(set.nodes)]
	set.mergeRoot(&set.Tree, &other.Tree, set.root, other.root, set.ul, other.ul, xor)
	return set
}

func (set IntervalSet) Difference(other IntervalSet) IntervalSet {
	set.nodes = set.nodes[:len(set.nodes)]
	set.mergeRoot(&set.Tree, &other.Tree, set.root, other.root, set.ul, !other.ul, and)
	return set
}

func (set IntervalSet) Equals(other IntervalSet) bool {
	return set.ul == other.ul && treeEquals(&set.Tree, &other.Tree, set.root, other.root)
}

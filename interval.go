package bandit

import (
	"errors"
	"fmt"
)

type (
	// Interval is an interval
	Interval struct {
		Tree
	}

	// BoundType is a bound type
	BoundType uint8
)

const (
	UnboundBound BoundType = iota
	ClosedBound
	OpenBound
)

const (
	empty    = "(Ø)"
	infinite = "(-∞, ∞)"
)

var (
	ErrInvalidInterval = errors.New("invalid interval")

	boundmap = map[bool]string{
		true:  "[)",
		false: "(]",
	}
)

func NewInterval(lowerBound BoundType, lower, upper uint64, upperBound BoundType) (ival Interval) {
	var (
		lul, rul    bool
		left, right uint32
	)
	ival.Tree.root = alloc_node()
	switch lowerBound {
	case UnboundBound:
		lul = true
	case OpenBound:
		left = ival.node(lower, 0, 0, 0, true, false)
	case ClosedBound:
		left = ival.node(lower, 0, 0, 0, true, true)
	}
	switch upperBound {
	case UnboundBound:
		rul = true
	case OpenBound:
		right = ival.node(upper, 0, 0, 0, true, true)
		rul = true
	case ClosedBound:
		right = ival.node(upper, 0, 0, 0, true, false)
		rul = true
	}
	ival.mergeRoot(&ival.Tree, &ival.Tree, left, right, lul, rul, and)
	return ival
}

func (ival Interval) Span() uint64 {
	lidx, _ := ival.Tree.leftmostLeaf(ival.root, ival.ul)
	ridx, _ := ival.Tree.rightmostLeaf(ival.root, ival.ul)
	return node_from_id(ridx).prefix - node_from_id(lidx).prefix
}

func (ival Interval) Lower() uint64 {
	idx, _ := ival.Tree.leftmostLeaf(ival.root, ival.ul)
	return node_from_id(idx).prefix
}

func (ival Interval) Upper() uint64 {
	idx, _ := ival.Tree.rightmostLeaf(ival.root, ival.ul)
	return node_from_id(idx).prefix
}

func (ival Interval) AsIntervalSet() *IntervalSet {
	return &IntervalSet{
		Tree: ival.Tree,
	}
}

func (ival Interval) String() string {
	if ival.root == 0 {
		if ival.ul {
			return infinite
		}
		return empty
	}
	n := node_from_id(ival.root)
	if n.level == 0 {
		// Half-unbounded interval or point
		if n.boundBoth() {
			return fmt.Sprintf("[%d]", n.prefix)
		}
		bs := boundmap[n.incl]
		if ival.ul {
			return fmt.Sprintf("(-∞, %d%c", n.prefix, bs[1])
		}
		return fmt.Sprintf("%c%d, ∞)", bs[0], n.prefix)
	}
	l, r := node_from_id(n.left), node_from_id(n.right)
	lb, rb := boundmap[l.incl][0], boundmap[r.incl][1]
	return fmt.Sprintf("%c%d, %d%c", lb, l.prefix, r.prefix, rb)
}

func (ival Interval) Intersection(other Interval) Interval {
	// Update the slice to point to the array copy
	//ival.nodes = ival.array[:len(ival.nodes)]
	// Merge onto the copy
	ival.mergeRoot(&ival.Tree, &other.Tree, ival.root, other.root, ival.ul, other.ul, and)
	return ival
}

func (ival Interval) Union(other Interval) Interval {
	// Update the slice to point to the array copy
	//ival.nodes = ival.array[:len(ival.nodes)]
	// Merge onto the copy
	ival.mergeRoot(&ival.Tree, &other.Tree, ival.root, other.root, ival.ul, other.ul, or)
	return ival
}

func (ival Interval) IsEmpty() bool {
	return ival.root == 0 && !ival.ul
}

func (ival Interval) Equals(other Interval) bool {
	return ival.ul == other.ul && treeEquals(&ival.Tree, &other.Tree, ival.root, other.root)
}

func LeftOpen(lower, upper uint64) Interval {
	return NewInterval(OpenBound, lower, upper, ClosedBound)
}

func RightOpen(lower, upper uint64) Interval {
	return NewInterval(ClosedBound, lower, upper, OpenBound)
}

func Closed(lower, upper uint64) Interval {
	return NewInterval(ClosedBound, lower, upper, ClosedBound)
}

func Point(val uint64) Interval {
	return Closed(val, val)
}

func Open(lower, upper uint64) Interval {
	return NewInterval(OpenBound, lower, upper, OpenBound)
}

func Above(value uint64) Interval {
	return NewInterval(OpenBound, value, 0, UnboundBound)
}

func AtOrAbove(value uint64) Interval {
	return NewInterval(ClosedBound, value, 0, UnboundBound)
}

func Below(value uint64) Interval {
	return NewInterval(UnboundBound, 0, value, OpenBound)
}

func AtOrBelow(value uint64) Interval {
	return NewInterval(UnboundBound, 0, value, ClosedBound)
}

func Empty() (ival Interval) {
	return NewInterval(OpenBound, 0, 0, OpenBound)
}

func Unbounded() (ival Interval) {
	return NewInterval(UnboundBound, 0, 0, UnboundBound)
}

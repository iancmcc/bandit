package bandit

import (
	"errors"
	"fmt"
)

type (
	// Interval is an interval
	Interval struct {
		Tree
		array [7]node
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

func NewInterval(lowerBound BoundType, lower, upper uint64, upperBound BoundType) Interval {
	var ival Interval
	ival.nodes = ival.array[:1] // Fix malloc later; for now we're escaping
	var (
		lul, rul    bool
		left, right uint
	)
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

func (ival Interval) String() string {
	if ival.root == 0 {
		if ival.ul {
			return infinite
		}
		return empty
	}
	n := ival.nodes[ival.root]
	if n.level == 0 {
		// Half-unbounded interval
		bs := boundmap[n.incl]
		if ival.ul {
			return fmt.Sprintf("(-∞, %d%c", n.prefix, bs[1])
		}
		return fmt.Sprintf("%c%d, ∞)", bs[0], n.prefix)
	}
	l, r := ival.nodes[n.left], ival.nodes[n.right]
	lb, rb := boundmap[l.incl][0], boundmap[r.incl][1]
	return fmt.Sprintf("%c%d, %d%c", lb, l.prefix, r.prefix, rb)
}

func (ival Interval) Intersection(other Interval) Interval {
	// Update the slice to point to the array copy
	ival.nodes = ival.array[:len(ival.nodes)]
	// Merge onto the copy
	ival.mergeRoot(&ival.Tree, &other.Tree, ival.root, other.root, ival.ul, other.ul, and)
	return ival
}

func (ival Interval) Union(other Interval) Interval {
	// Update the slice to point to the array copy
	ival.nodes = ival.array[:len(ival.nodes)]
	// Merge onto the copy
	ival.mergeRoot(&ival.Tree, &other.Tree, ival.root, other.root, ival.ul, other.ul, or)
	return ival
}

func (ival Interval) IsEmpty() bool {
	return ival.root == 0 && !ival.ul
}

func (ival Interval) Equals(other Interval) bool {
	if ival.ul != other.ul {
		return false
	}
	a, b := ival.nodes[ival.root], other.nodes[other.root]
	return (a.Equals(b) &&
		ival.nodes[a.left].Equals(other.nodes[b.left]) &&
		ival.nodes[a.right].Equals(other.nodes[b.right]))
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

package bandit

import (
	"errors"
)

type (
	// Interval is an interval
	Interval struct {
		Tree
		array [4]node
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
	inf     = '∞'
	lclosed = '['
	rclosed = ']'
	lopen   = '('
	ropen   = ')'
)

var (
	ErrInvalidInterval = errors.New("invalid interval")
)

func (ival *Interval) SetInterval(lowerBound BoundType, lower, upper uint64, upperBound BoundType) *Interval {
	ival.nodes = ival.array[:1]
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

func (ival *Interval) Intersection(other *Interval) *Interval {
	ival.mergeRoot(&ival.Tree, &other.Tree, ival.root, other.root, ival.ul, other.ul, and)
	return ival
}

func (ival *Interval) Union(other *Interval) *Interval {
	ival.mergeRoot(&ival.Tree, &other.Tree, ival.root, other.root, ival.ul, other.ul, or)
	return ival
}

func (ival *Interval) SymmetricDifference(other *Interval) *Interval {
	ival.mergeRoot(&ival.Tree, &other.Tree, ival.root, other.root, ival.ul, other.ul, xor)
	return ival
}

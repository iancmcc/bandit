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
	ival.nodes = ival.array[0:1]
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

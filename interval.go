package bandit

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
	UnboundBound = iota
	ClosedBound
	OpenBound
)

/*
func set(ival *Interval, at, bt *Tree, lower, upper node, lul, rul bool) {
	ival.nodes = ival.array[0:2]
	ival.root = ival.merge(at, bt, lower, upper, lul, rul, and)
}
*/

func Set(ival *Interval, lowerBound BoundType, lower, upper uint64, upperBound BoundType) {
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
	ival.root = ival.merge(&ival.Tree, &ival.Tree, left, right, lul, rul, and)
}

func (ival Interval) String() string {
	return "hi"
}

package bandit_test

import (
	"testing"

	. "github.com/iancmcc/bandit/ng"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBranchingBit(t *testing.T) {

	var a uint64

	Convey("Branching bit prefixes", t, func() {
		var b uint64
		for i := 0; i < 64; i++ {
			b |= 1 << i
			So(BranchingBit(a, b), ShouldEqual, i+1)
		}
	})
}

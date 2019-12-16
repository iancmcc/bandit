package bandit_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	. "github.com/iancmcc/bandit"
)

var I = MustParseIntervalString

var _ = Describe("Interval", func() {

	DescribeTable("parsing intervals",
		func(s string, lowerBound BoundType, lower, upper int, upperBound BoundType) {
			expected := NewInterval(lowerBound, uint64(lower), uint64(upper), upperBound)

			ival, err := ParseIntervalString(s)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ival).Should(Equal(expected))

			ival, err = ParseInterval([]byte(s))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ival).Should(Equal(expected))
		},
		Entry("()", "(11, 29)", OpenBound, 11, 29, OpenBound),
		Entry("[]", "[12, 28]", ClosedBound, 12, 28, ClosedBound),
		Entry("(]", "(13, 27]", OpenBound, 13, 27, ClosedBound),
		Entry("[)", "[14, 26)", ClosedBound, 14, 26, OpenBound),
		Entry("unbound", "(-inf, inf)", UnboundBound, 0, 0, UnboundBound),
	)

	Context("performing intersection", func() {
		It("should return the smaller interval when one is fully contained by the other", func() {
			a := I("(1, 100]")
			b := I("[25, 30)")
			c := a.Intersection(b)
			Ω(c.Equals(b)).Should(BeTrue())
		})

		It("should return an empty interval when the two intervals are disjoint", func() {
			a := I("(0, 100]")
			b := I("[200, 300]")
			c := a.Intersection(b)
			Ω(c.IsEmpty()).Should(BeTrue())

			d := b.Intersection(a)
			Ω(d.IsEmpty()).Should(BeTrue())
		})

		It("should return an identical interval when the two intervals are equal", func() {
			a := I("(0, 100]")
			b := I("(0, 100]")
			c := a.Intersection(b)
			Ω(c.Equals(a)).Should(BeTrue())
			Ω(c.Equals(b)).Should(BeTrue())
		})

		It("should return the correct intersection when the intersecting interval overlaps on the left", func() {
			a := LeftOpen(50, 100)
			b := RightOpen(40, 60)
			c := a.Intersection(b)
			expected := Open(50, 60)
			Ω(c.Equals(expected)).Should(BeTrue())
		})

		It("should return the correct intersection when the intersecting interval overlaps on the right", func() {
			a := LeftOpen(50, 100)
			b := RightOpen(90, 110)
			c := a.Intersection(b)
			expected := Closed(90, 100)
			Ω(c.Equals(expected)).Should(BeTrue())
		})

		It("should return the smaller when the two intervals have an identical lower bound", func() {
			a := LeftOpen(50, 100)
			b := LeftOpen(50, 60)
			c := a.Intersection(b)
			Ω(c.Equals(b)).Should(BeTrue())
		})

		It("should return the smaller when the two intervals have an identical upper bound", func() {
			a := Closed(50, 100)
			b := Closed(60, 100)
			c := a.Intersection(b)
			Ω(c.Equals(b)).Should(BeTrue())
		})

		It("should return a left-open interval when the lower bound differs only in openness", func() {
			a := Open(10, 100)
			b := Closed(10, 50)
			c := a.Intersection(b)
			expected := LeftOpen(10, 50)
			Ω(c.Equals(expected)).Should(BeTrue())
		})

		It("should return a right-open interval when the upper bound differs only in openness", func() {
			a := Open(0, 100)
			b := Closed(10, 100)
			c := a.Intersection(b)
			expected := RightOpen(10, 100)
			Ω(c.Equals(expected)).Should(BeTrue())
		})

		It("shouldn't mutate the original interval", func() {
			last := Open(1, 10)
			next := last.Intersection(AtOrAbove(2))
			Ω(last).Should(Equal(Open(1, 10)))
			Ω(next).Should(Equal(RightOpen(2, 10)))
		})

		It("should return representative strings", func() {
			Ω(Open(100, 200).String()).Should(Equal("(100, 200)"))
			Ω(Closed(100, 200).String()).Should(Equal("[100, 200]"))
			Ω(LeftOpen(100, 200).String()).Should(Equal("(100, 200]"))
			Ω(RightOpen(100, 200).String()).Should(Equal("[100, 200)"))
			Ω(Open(0, 0).String()).Should(Equal("(Ø)"))
			Ω(Below(10).String()).Should(Equal("(-∞, 10)"))
			Ω(AtOrBelow(10).String()).Should(Equal("(-∞, 10]"))
			Ω(Above(10).String()).Should(Equal("(10, ∞)"))
			Ω(AtOrAbove(10).String()).Should(Equal("[10, ∞)"))
		})

		It("should report as empty if the bounds are equal and at least one is open", func() {
			Ω(Open(0, 0).IsEmpty()).Should(BeTrue())
			Ω(Closed(0, 0).IsEmpty()).Should(BeFalse())
			Ω(LeftOpen(10, 10).IsEmpty()).Should(BeTrue())
			Ω(RightOpen(10, 10).IsEmpty()).Should(BeTrue())
			Ω(Open(10, 11).IsEmpty()).Should(BeFalse())
		})

		DescribeTable("It should intersect unbounded intervals correctly", func(lhs, rhs, expected Interval) {
			intersection := lhs.Intersection(rhs)
			Ω(intersection.Equals(expected)).Should(BeTrue(), fmt.Sprintf("Expected %s to equal %s", &intersection, &expected))
			intersection = rhs.Intersection(lhs)
			Ω(intersection.Equals(expected)).Should(BeTrue(), fmt.Sprintf("Expected %s to equal %s", &intersection, &expected))
		},
			Entry("one lower unbounded, other fully encompassed", Below(10), Closed(0, 5), Closed(0, 5)),
			Entry("one lower unbounded, other intersects", Below(10), Closed(0, 50), RightOpen(0, 10)),
			Entry("one lower unbounded, other no overlap", Below(10), Closed(10, 20), Empty()),
			Entry("one lower unbounded, other empty", Below(10), Empty(), Empty()),
			Entry("one lower unbounded, other lower unbounded", Below(10), AtOrBelow(20), Below(10)),
			Entry("one lower unbounded, other upper unbounded and intersects", Below(10), Above(5), Open(5, 10)),
			Entry("one lower unbounded, other upper unbounded, no intersection", Below(10), Above(10), Empty()),
			Entry("one upper unbounded, other fully encompassed", Above(10), Closed(15, 20), Closed(15, 20)),
			Entry("one upper unbounded, other intersects", Above(10), Closed(0, 50), LeftOpen(10, 50)),
			Entry("one upper unbounded, other no overlap", Above(10), Closed(0, 5), Empty()),
			Entry("one upper unbounded, other empty", Above(10), Empty(), Empty()),
			Entry("one fully unbounded, other bounded", Unbounded(), Closed(0, 5), Closed(0, 5)),
			Entry("one fully unbounded, other left unbounded", Unbounded(), Below(10), Below(10)),
			Entry("one fully unbounded, other right unbounded", Unbounded(), Above(10), Above(10)),
			Entry("one fully unbounded, other empty", Unbounded(), Empty(), Empty()),
		)

	})

	Context("Comparing Intervals", func() {
		DescribeTable("It should report equality correctly", func(intv1, intv2 Interval, eq bool) {
			s := "equal"
			if !eq {
				s = "not equal"
			}
			Ω(intv1.Equals(intv2)).Should(Equal(eq), fmt.Sprintf("Expected %s to %s %s", intv1, s, intv2))
			Ω(intv2.Equals(intv1)).Should(Equal(eq), fmt.Sprintf("Expected %s to %s %s", intv2, s, intv1))
		},
			Entry("Identical closed", NewInterval(ClosedBound, 1, 8, ClosedBound), Closed(1, 8), true),
			Entry("Different closed - upper", NewInterval(ClosedBound, 1, 9, ClosedBound), Closed(1, 8), false),
			Entry("Different closed - lower", NewInterval(ClosedBound, 2, 8, ClosedBound), Closed(1, 8), false),
			Entry("Identical left-open", NewInterval(OpenBound, 1, 8, ClosedBound), LeftOpen(1, 8), true),
			Entry("Different left-open - upper", NewInterval(OpenBound, 1, 9, ClosedBound), LeftOpen(1, 8), false),
			Entry("Different left-open - lower", NewInterval(OpenBound, 2, 8, ClosedBound), LeftOpen(1, 8), false),
			Entry("Identical right-open", NewInterval(ClosedBound, 1, 8, OpenBound), RightOpen(1, 8), true),
			Entry("Different right-open - upper", NewInterval(ClosedBound, 1, 9, OpenBound), RightOpen(1, 8), false),
			Entry("Different right-open - lower", NewInterval(ClosedBound, 2, 8, OpenBound), RightOpen(1, 8), false),
			Entry("Identical left-unbounded", NewInterval(UnboundBound, 1, 8, OpenBound), Below(8), true),
			Entry("Different left-unbounded", NewInterval(UnboundBound, 1, 9, OpenBound), Below(8), false),
			Entry("Identical right-unbounded", NewInterval(OpenBound, 1, 8, UnboundBound), Above(1), true),
			Entry("Different right-unbounded", NewInterval(OpenBound, 2, 9, UnboundBound), Above(1), false),
			Entry("Both unbounded", NewInterval(UnboundBound, 2, 9, UnboundBound), Unbounded(), true),
			Entry("Both empty", NewInterval(ClosedBound, 2, 2, OpenBound), Empty(), true),
			Entry("Closed v Open", Open(1, 8), Closed(1, 8), false),
			Entry("LeftOpen v RightOpen", LeftOpen(1, 8), RightOpen(1, 8), false),
			Entry("Bounded v right-unbounded", NewInterval(OpenBound, 2, 9, UnboundBound), NewInterval(OpenBound, 2, 9, ClosedBound), false),
			Entry("Bounded v left-unbounded", NewInterval(UnboundBound, 2, 9, ClosedBound), NewInterval(OpenBound, 2, 9, ClosedBound), false),
			Entry("Bounded v unbounded", NewInterval(UnboundBound, 2, 9, UnboundBound), NewInterval(OpenBound, 2, 9, ClosedBound), false),
		)
	})

})

package bandit_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	. "github.com/iancmcc/bandit"
)

func doIntervalSetOp(op string, a, b *IntervalSet) *IntervalSet {
	switch op {
	case "&":
		return a.Intersection(a, b)
	case "|":
		return a.Union(a, b)
	case "^":
		return a.SymmetricDifference(a, b)
	case "-":
		return a.Difference(a, b)
	}
	return a
}

var _ = Describe("Set", func() {
	var a, b *IntervalSet

	BeforeEach(func() {
		a = NewIntervalSet(RightOpen(0, 2), RightOpen(4, 6))
		b = NewIntervalSet(RightOpen(1, 3), RightOpen(3, 5))
	})

	DescribeTable("interval set operations",
		func(op, expected string) {
			Ω(doIntervalSetOp(op, a, b).String()).Should(Equal(expected))
		},
		Entry("a & b", "&", `[1, 2), [4, 5)`),
		Entry("a | b", "|", `[0, 6)`),
		Entry("a - b", "-", `[0, 1), [5, 6)`),
		Entry("a ^ b", "^", `[0, 1), [2, 4), [5, 6)`),
	)

	It("should find enclosing intervals", func() {
		a := NewIntervalSet(RightOpen(1, 10), RightOpen(20, 30), RightOpen(40, 50))
		b := NewIntervalSet(RightOpen(4, 5), RightOpen(19, 25), RightOpen(42, 49))
		c := NewIntervalSet().Enclosed(a, b)
		Ω(c.Equals(NewIntervalSet(RightOpen(4, 5), RightOpen(42, 49)))).Should(BeTrue())
	})

	It("should find common intervals", func() {

		Ω(NewIntervalSet().CommonIntervals(a, b).IsEmpty()).Should(BeTrue())

		a.Add(a,
			//AtOrBelow(2), // FIXME: Left unbounded common is broken
			RightOpen(7, 9),
			RightOpen(10, 13),
			Closed(14, 14),
			Closed(15, 20),
			LeftOpen(22, 27),
			Above(30))

		b.Add(b,
			//AtOrBelow(2), // FIXME: Left unbounded common is broken
			RightOpen(7, 9),
			RightOpen(10, 12),
			Closed(14, 14),
			Closed(15, 20),
			LeftOpen(23, 27),
			Above(30))

		actual := b.CommonIntervals(a, b)
		expected := NewIntervalSet(RightOpen(7, 9), Closed(14, 14), Closed(15, 20), Above(30))

		Ω(actual.Equals(expected)).Should(BeTrue(), "%s != %s", actual, expected)
	})

	It("should report equality correctly", func() {
		Ω(a.Equals(b)).Should(BeFalse())
		a.Union(a, b)
		b.Union(b, a)
		Ω(a.Equals(b)).Should(BeTrue())
	})

	It("should find the complement correctly", func() {
		Ω(a.Complement(a).String()).Should(Equal(`(-∞, 0), [2, 4), [6, ∞)`))
	})

	It("should find containing intervals", func() {
		ival := a.IntervalContaining(1)
		Ω(ival.Equals(RightOpen(0, 2))).Should(BeTrue())
	})

	It("should return the interval containing a point", func() {
		// (-∞, 2) (2, 4] [5, 10] (15, 17), (17, ∞)
		ivals := []Interval{
			Empty(),
			Below(2),
			LeftOpen(2, 4),
			Closed(5, 10),
			Open(15, 17),
			Above(17),
		}
		a = NewIntervalSet(ivals...)

		expected := []int{1, 1, 0, 2, 2, 3, 3, 3, 3, 3, 3, 0, 0, 0, 0, 0, 4, 0, 5, 5, 5}

		for i := 0; i < len(expected); i++ {
			actual := a.IntervalContaining(uint64(i))
			exp := ivals[expected[i]]
			Ω(actual.Equals(exp)).Should(BeTrue(), fmt.Sprintf("%s != %s", actual, exp))
		}

		// Get rid of the unbounded sides to test bounded tries
		a.Intersection(a, NewIntervalSet(Closed(1, 19)))

		// Set new expectations
		expected[0] = 0
		expected[20] = 0
		ivals[1] = RightOpen(1, 2)
		ivals[5] = LeftOpen(17, 19)

		for i := 0; i < len(expected); i++ {
			actual := a.IntervalContaining(uint64(i))
			exp := ivals[expected[i]]
			Ω(actual.Equals(exp)).Should(BeTrue(), fmt.Sprintf("%s != %s", actual, exp))
		}
	})

})

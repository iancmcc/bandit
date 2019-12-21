package bandit_test

import (
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

	It("should report equality correctly", func() {
		Ω(a.Equals(b)).Should(BeFalse())
		a.Union(a, b)
		b.Union(b, a)
		Ω(a.Equals(b)).Should(BeTrue())
	})

	/*
		It("should find the complement correctly", func() {
			Ω(a.Complement().String()).Should(Equal(`(-∞, 0), [2, 4), [6, ∞)`))
		})
	*/

})

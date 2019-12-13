package bandit_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	. "github.com/iancmcc/bandit"
)

var _ = Describe("Interval", func() {

	DescribeTable("parsing intervals",
		func(s string, lowerBound BoundType, lower, upper int, upperBound BoundType) {
			var ival, expected Interval
			expected.SetInterval(lowerBound, uint64(lower), uint64(upper), upperBound)

			_, err := ival.ParseIntervalString(s)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ival).Should(Equal(expected))

			_, err = ival.ParseInterval([]byte(s))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ival).Should(Equal(expected))
		},
		Entry("()", "(11, 29)", OpenBound, 11, 29, OpenBound),
		Entry("[]", "[12, 28]", ClosedBound, 12, 28, ClosedBound),
		Entry("(]", "(13, 27]", OpenBound, 13, 27, ClosedBound),
		Entry("[)", "[14, 26)", ClosedBound, 14, 26, OpenBound),
		Entry("unbound", "(-inf, inf)", UnboundBound, 0, 0, UnboundBound),
	)

	It("should intersect", func() {
		var a, b Interval
		a.MustParseIntervalString("(10, 20)")
		b.MustParseIntervalString("(15, 25)")

		c := a.Intersection(&b)

		Ω(c).Should(Equal(new(Interval).MustParseIntervalString("(15, 20)")))
	})

})

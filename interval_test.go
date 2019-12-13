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
			err := ival.ParseIntervalString(s)
			Ω(err).ShouldNot(HaveOccurred())
			expected.SetInterval(lowerBound, uint64(lower), uint64(upper), upperBound)
			Ω(ival.String()).Should(Equal(expected.String()))
		},
		Entry("()", "(11, 29)", OpenBound, 11, 29, OpenBound),
		Entry("[]", "[12, 28]", ClosedBound, 12, 28, ClosedBound),
		Entry("(]", "(13, 27]", OpenBound, 13, 27, ClosedBound),
		Entry("[)", "[14, 26)", ClosedBound, 14, 26, OpenBound),
		Entry("unbound", "(-inf, inf)", UnboundBound, 0, 0, UnboundBound),
	)

})

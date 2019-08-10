package bandit_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/iancmcc/bandit"
)

func runit() {
	var ival Interval
	Set(&ival, ClosedBound, 10, 20, OpenBound)
	Ω(ival.String()).Should(BeTrue())
}

var _ = Describe("Interval", func() {

	/*
		It("should do things right", func() {
			ival, err := Uint("hi")
			Ω(err).Should(MatchError(ErrInvalidInterval))

			ival, err = Uint("[10, 20]")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(ival.IsEmpty()).Should(BeFalse())
			Ω(ival.String()).Should(Equal("[10, 20]"))

			ival, err = Uint("(-∞, 20]")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ival.String()).Should(Equal("(-∞, 20]"))

			ival, err = Int("(-10, 20)")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ival.String()).Should(Equal("(-10, 20)"))

			ival, err = Float("(-10.1, 20.3345)")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ival.String()).Should(Equal("(-10.1, 20.3345)"))
		})

	*/

	It("should go", runit)
})

package bandit_test

import (
	"bytes"

	. "github.com/iancmcc/bandit"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tree", func() {

	It("should dump and load", func() {
		var buf bytes.Buffer

		// (-∞, 2) (2, 4] [5, 10] (15, 17), (17, ∞)
		ivals := []Interval{
			Empty(),
			Below(2),
			LeftOpen(2, 4),
			Closed(5, 10),
			Open(15, 17),
			Above(17),
		}
		orig := NewIntervalSet(ivals...)

		Ω(orig.Tree.Dump(&buf)).ShouldNot(HaveOccurred())

		//fmt.Println(base64.StdEncoding.EncodeToString(buf.Bytes()))

		decoded, err := LoadTree(&buf)
		Ω(err).ShouldNot(HaveOccurred())

		newset := IntervalSet{*decoded}

		Ω(newset.Equals(orig)).Should(BeTrue())

	})

})

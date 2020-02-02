package bandit_test

import (
	"golang.org/x/exp/rand"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	. "github.com/iancmcc/bandit"
)

func doIntervalMapOp(op string, a, b *IntervalMap) *IntervalMap {
	switch op {
	case "&":
		return a.Intersection(a, b)
	case "|":
		return a.Union(a, b)
	case "^":
		return a.SymmetricDifference(a, b)
	case "-":
		return a.Difference(a, b)
	case ">":
		return a.Enclosed(a, b)
	}
	return a
}

func imap(v interface{}, ivals ...Interval) *IntervalMap {
	return NewIntervalMapWithCapacity(1, 4).Add(nil, v, ivals...)
}

var _ = Describe("IntervalMap", func() {

	var (
		b = NewIntervalMapWithCapacity(1, 2).Add(nil, "", AtOrAbove(1))
		c = NewIntervalMapWithCapacity(1, 2).Add(nil, "", Closed(1, 1))
		d = NewIntervalMapWithCapacity(1, 2).Add(nil, "", Below(1), Above(1))
	)

	DescribeTable("interval map operations",
		func(op string, other, expected *IntervalMap) {
			a := imap("", Above(1))
			actual := doIntervalMapOp(op, a, other)
			Ω(actual.Equals(expected)).Should(BeTrue(), "%s != %s", actual.Get(""), expected.Get(""))
		},
		Entry("(1, ∞) | [1, ∞)", "|", b, imap("", AtOrAbove(1))),
		Entry("(1, ∞) & [1, ∞)", "&", b, imap("", Above(1))),
		Entry("(1, ∞) ^ [1, ∞)", "^", b, imap("", Point(1))),
		Entry("(1, ∞) - [1, ∞)", "-", b, imap("", Empty())),
		Entry("(1, ∞) | [1]", "|", c, imap("", AtOrAbove(1))),
		Entry("(1, ∞) & [1]", "&", c, imap("", Empty())),
		Entry("(1, ∞) ^ [1]", "^", c, imap("", AtOrAbove(1))),
		Entry("(1, ∞) - [1]", "-", c, imap("", Above(1))),
		Entry("(1, ∞) | ]1[", "|", d, imap("", Below(1), Above(1))),
		Entry("(1, ∞) & ]1[", "&", d, imap("", Above(1))),
		Entry("(1, ∞) ^ ]1[", "^", d, imap("", Below(1))),
		Entry("(1, ∞) - ]1[", "-", d, imap("", Empty())),
	)

	operatorTest := func(astr string, operator string, bstr string, expectedstr ...string) {
		ivals := make([]Interval, len(expectedstr))
		for i, s := range expectedstr {
			ivals[i] = MustParseIntervalString(s)
		}
		expected := imap("", ivals...)
		left := imap("", MustParseIntervalString(astr))
		right := imap("", MustParseIntervalString(bstr))

		actual := doIntervalMapOp(operator, left, right)
		Ω(actual.Equals(expected)).Should(BeTrue(), "%s != %s", actual.Get(""), expected.Get(""))

		left = imap("", MustParseIntervalString(astr))
		right = imap("", MustParseIntervalString(bstr))

		// Everything except Difference and Enclosed is a commutative
		// operation, so test the other direction
		if operator != "-" && operator != ">" {
			inverse := doIntervalMapOp(operator, right, left)
			Ω(inverse.Equals(expected)).Should(BeTrue())
		}
	}

	DescribeTable("(a, b) & (c, d)", operatorTest,
		Entry("a < b < c < d", "(100, 200)", "&", "(300, 400)", "(0, 0)"),
		Entry("a < b = c < d", "(100, 200)", "&", "(200, 300)", "(0, 0)"),
		Entry("a < c < b < d", "(100, 300)", "&", "(200, 400)", "(200, 300)"),
		Entry("a < c < d < b", "(100, 400)", "&", "(200, 300)", "(200, 300)"),
		Entry("a = c < b = d", "(100, 200)", "&", "(100, 200)", "(100, 200)"),
		Entry("a = c < d < b", "(100, 400)", "&", "(100, 300)", "(100, 300)"),
		Entry("a < c < b = d", "(100, 400)", "&", "(300, 400)", "(300, 400)"),
		Entry("(c, d) = Ø", "(100, 400)", "&", "(1, 0)", "(0, 0)"),
		Entry("(a, b) = Ø", "(1, 0)", "&", "(100, 400)", "(0, 0)"),
		Entry("(a, b) = Ø, (c, d) = Ø", "(1, 0)", "&", "(1, 0)", "(0, 0)"),
	)

	DescribeTable("(a, b) | (c, d)", operatorTest,
		Entry("a < b < c < d", "(100, 200)", "|", "(300, 400)", "(100, 200)", "(300, 400)"),
		Entry("a < b = c < d", "(100, 200)", "|", "(200, 300)", "(100, 200)", "(200, 300)"),
		Entry("a < c < b < d", "(100, 300)", "|", "(200, 400)", "(100, 400)"),
		Entry("a < c < d < b", "(100, 400)", "|", "(200, 300)", "(100, 400)"),
		Entry("a = c < b = d", "(100, 200)", "|", "(100, 200)", "(100, 200)"),
		Entry("a = c < d < b", "(100, 400)", "|", "(100, 300)", "(100, 400)"),
		Entry("a < c < b = d", "(100, 400)", "|", "(300, 400)", "(100, 400)"),
		Entry("(c, d) = Ø", "(100, 400)", "|", "(1, 0)", "(100, 400)"),
		Entry("(a, b) = Ø", "(1, 0)", "|", "(100, 400)", "(100, 400)"),
		Entry("(a, b) = Ø, (c, d) = Ø", "(1, 0)", "|", "(1, 0)", "(0, 0)"),
	)

	DescribeTable("(a, b) ^ (c, d)", operatorTest,
		Entry("a < b < c < d", "(100, 200)", "^", "(300, 400)", "(100, 200)", "(300, 400)"),
		Entry("a < b = c < d", "(100, 200)", "^", "(200, 300)", "(100, 200)", "(200, 300)"),
		Entry("a < c < b < d", "(100, 300)", "^", "(200, 400)", "(100, 200]", "[300, 400)"),
		Entry("a < c < d < b", "(100, 400)", "^", "(200, 300)", "(100, 200]", "[300, 400)"),
		Entry("a = c < b = d", "(100, 200)", "^", "(100, 200)", "(0, 0)"),
		Entry("a = c < d < b", "(100, 400)", "^", "(100, 300)", "[300, 400)"),
		Entry("a < c < b = d", "(100, 400)", "^", "(300, 400)", "(100, 300]"),
		Entry("(c, d) = Ø", "(100, 400)", "^", "(0, 0)", "(100, 400)"),
		Entry("(a, b) = Ø", "(1, 0)", "^", "(100, 400)", "(100, 400)"),
		Entry("(a, b) = Ø, (c, d) = Ø", "(1, 0)", "^", "(1, 0)", "(0, 0)"),
	)

	DescribeTable("(a, b) &^ (c, d)", operatorTest,
		Entry("a < b < c < d", "(100, 200)", "-", "(300, 400)", "(100, 200)"),
		Entry("a < b = c < d", "(100, 200)", "-", "(200, 300)", "(100, 200)"),
		Entry("a < c < b < d", "(100, 300)", "-", "(200, 400)", "(100, 200]"),
		Entry("a < c < d < b", "(100, 400)", "-", "(200, 300)", "(100, 200]", "[300, 400)"),
		Entry("a = c < b = d", "(100, 200)", "-", "(100, 200)", "(0, 0)"),
		Entry("a = c < d < b", "(100, 400)", "-", "(100, 300)", "[300, 400)"),
		Entry("a < c < b = d", "(100, 400)", "-", "(300, 400)", "(100, 300]"),
		Entry("(c, d) = Ø", "(100, 400)", "-", "(1, 0)", "(100, 400)"),
		Entry("(a, b) = Ø", "(1, 0)", "-", "(100, 400)", "(0, 0)"),
		Entry("(a, b) = Ø, (c, d) = Ø", "(1, 0)", "-", "(1, 0)", "(0, 0)"),
	)

	DescribeTable("(a, b) > (c, d)", operatorTest,
		Entry("a < b < c < d", "(100, 200)", ">", "(300, 400)", "(0, 0)"),
		Entry("a < b = c < d", "(100, 200)", ">", "(200, 300)", "(0, 0)"),
		Entry("a < c < b < d", "(100, 300)", ">", "(200, 400)", "(0, 0)"),
		Entry("a < c < d < b", "(100, 400)", ">", "(200, 300)", "(200, 300)"),
		Entry("a = c < b = d", "(100, 200)", ">", "(100, 200)", "(100, 200)"),
		Entry("a = c < d < b", "(100, 400)", ">", "(100, 300)", "(100, 300)"),
		Entry("a < c < b = d", "(100, 400)", ">", "(300, 400)", "(300, 400)"),
		Entry("(c, d) = Ø", "(100, 400)", ">", "(0, 0)", "(0, 0)"),
		Entry("(a, b) = Ø", "(0, 0)", ">", "(100, 400)", "(0, 0)"),
		Entry("(a, b) = Ø, (c, d) = Ø", "(0, 0)", ">", "(0, 0)", "(0, 0)"),
	)

	It("should pick a random value", func() {

		m := NewIntervalMap()
		m.Add(m, "a", RightOpen(10, 20))
		m.Add(m, "b", RightOpen(10, 20), RightOpen(30, 40))
		m.Add(m, "c", RightOpen(10, 20), RightOpen(30, 40), RightOpen(50, 60))

		counts := map[interface{}]map[string]int{
			"a": make(map[string]int),
			"b": make(map[string]int),
			"c": make(map[string]int),
		}

		rng := rand.New(rand.NewSource(uint64(GinkgoRandomSeed())))

		for i := 0; i < 6000; i++ {
			v, ival := m.RandValue(rng, 1)
			counts[v][ival.String()] += 1
		}
		for _, m := range counts {
			for _, c := range m {
				Ω(c).Should(BeNumerically("~", 1000, 100))
			}
		}

	})
})

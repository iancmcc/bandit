package bandit_test

import (
	"testing"

	. "github.com/iancmcc/bandit/ng"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNodes(t *testing.T) {

	bucket_size := 8

	Convey("a Nodes instance", t, func() {
		nodes := NewNodes(bucket_size)

		Convey("should allocate nodes", func() {
			a := nodes.Alloc()
			b := nodes.Alloc()
			c := nodes.Alloc()

			So(a, ShouldEqual, 1)
			So(b, ShouldEqual, 2)
			So(c, ShouldEqual, 3)

			Convey("and allow them to be retrieved", func() {

				nodes.Get(a).Prefix = 1
				nodes.Get(b).Prefix = 2

				So(nodes.Get(a).Prefix, ShouldEqual, 1)
				So(nodes.Get(b).Prefix, ShouldEqual, 2)
			})
		})

		Convey("should grow when needed", func() {
			for j := 1; j < bucket_size*3; j++ {
				So(nodes.Alloc(), ShouldEqual, j)
			}
		})

		Convey("should allow nodes to be freed", func() {
			a := nodes.Alloc()
			b := nodes.Alloc()
			nodes.Get(a).Prefix = 1
			nodes.Get(b).Prefix = 2

			nodes.Free(a)
			nodes.Free(b)

			Convey("and reuse freed nodes", func() {
				c := nodes.Alloc()
				So(c, ShouldEqual, b)

				d := nodes.Alloc()
				So(d, ShouldEqual, a)
			})

		})
	})

}

func BenchmarkNodes(b *testing.B) {
	n := NewNodes(1024)
	b.ReportAllocs()
	b.ResetTimer()
	for j := 0; j < b.N; j++ {
		n.Alloc()
		n.Free(n.Alloc())
	}
}

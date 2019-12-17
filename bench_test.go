package bandit

import (
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/zenoss/yamr/interval"
	"github.com/zenoss/yamr/temporalset"
)

func measureMemoryUsageDuringOperation(body func()) float64 {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	body()
	runtime.ReadMemStats(&m2)
	return float64(m2.Alloc-m1.Alloc) / 1024 / 1024
}

func createTestTrie(n, capacity, offset, stride int) IntervalSet {
	r := NewIntervalSet(Above(0))
	for i := 0; i < n; i++ {
		r = r.SymmetricDifference(NewIntervalSet(Above(uint64(i*stride + offset))))
	}
	return r
}

func createOldTestTrie(n, capacity, offset, stride int) *temporalset.IntervalSet {
	r := temporalset.NewIntervalSet(interval.Above(0))
	for i := 0; i < n; i++ {
		r = r.SymmetricDifference(temporalset.NewIntervalSet(interval.Above(uint64(i*stride + offset))))
	}
	return r
}

var _ = Describe("Bench", func() {

	var (
		n int = 800
		m int = 10000
	)

	Measure("or", func(bm Benchmarker) {
		var a, b IntervalSet
		creation := bm.Time("creation", func() {
			a = createTestTrie(n, m, 0, 2)
			b = createTestTrie(n, m, 1, 2)
		})
		Ω(creation.Seconds()).Should(BeNumerically("<", 15))
		runtime := bm.Time("runtime", func() {
			a.Union(b)
		})
		Ω(runtime.Seconds()).Should(BeNumerically("<", 2))

	}, 1000)

	/*
		Measure("or - old", func(bm Benchmarker) {
			var a, b *temporalset.IntervalSet
			alloc := measureMemoryUsageDuringOperation(func() {
				bm.Time("creation", func() {
					a = createOldTestTrie(n, m, 0, 2)
					b = createOldTestTrie(n, m, 1, 2)
				})
			})
			bm.RecordValue("creationAlloc", alloc)
			runtime := bm.Time("runtime", func() {
				a.Union(b)
			})
			Ω(runtime.Seconds()).Should(BeNumerically("<", 2))

		}, 1000)
	*/

	Measure("and", func(bm Benchmarker) {
		var a, b IntervalSet
		creation := bm.Time("creation", func() {
			a = createTestTrie(n, m, 0, 2)
			b = createTestTrie(n, m, 1, 2)
		})
		Ω(creation.Seconds()).Should(BeNumerically("<", 15))
		runtime := bm.Time("runtime", func() {
			a.Intersection(b)
		})
		Ω(runtime.Seconds()).Should(BeNumerically("<", 2))

	}, 1000)

	/*
		Measure("and - old", func(bm Benchmarker) {
			var a, b *temporalset.IntervalSet
			alloc := measureMemoryUsageDuringOperation(func() {
				bm.Time("creation", func() {
					a = createOldTestTrie(n, m, 0, 2)
					b = createOldTestTrie(n, m, 1, 2)
				})
			})
			bm.RecordValue("creationAlloc", alloc)
			runtime := bm.Time("runtime", func() {
				a.Intersection(b)
			})
			Ω(runtime.Seconds()).Should(BeNumerically("<", 2))

		}, 1000)
	*/

	/*
		Measure("and", func(bm Benchmarker) {
			values := make([]interface{}, m)
			for i := 0; i < m; i++ {
				values[i] = zenkit.RandString(20)
			}
			a := createTestTrie(n, m, 0, 2, values)
			b := createTestTrie(n, m, 1, 2, values)
			var c TemporalSet
			runtime := bm.Time("runtime", func() {
				c = a.And(b)
				var count int64
				c.Each(func(i interval.Interval, v interface{}) {
					count++
				})
				Ω(count).Should(BeNumerically("==", n*m/2))
			})
			Ω(runtime.Seconds()).Should(BeNumerically("<", 2))

		}, 1)

		Measure("walking intervals", func(bm Benchmarker) {
			values := make([]interface{}, m)
			for i := 0; i < m; i++ {
				values[i] = zenkit.RandString(20)
			}
			do := func(n, span int) {
				s := EmptyWithCapacity(span)
				for i := 0; i < span; i++ {
					s = s.Or(createTestTrie(n, span, i, span, []interface{}{zenkit.RandString(8)}))
				}
				bm.Time(fmt.Sprintf("%d values * %d intervals", span, n), func() {
					var count int64
					s.EachInterval(func(i interval.Interval, v []interface{}) {
						count++
					})
					Ω(count).Should(BeNumerically("==", (n*span)-((n-1)/2)-1))
				})

			}
			do(10000, 3)
			do(3, 10000)
		}, 1)
	*/

})

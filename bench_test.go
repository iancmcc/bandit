package bandit

import (
	"runtime"
	"runtime/debug"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/zenoss/yamr/interval"
	"github.com/zenoss/yamr/temporalset"
)

func measureMemoryUsageDuringOperation(bm Benchmarker, prefix string, body func()) float64 {
	var m1, m2 runtime.MemStats
	bm.Time(prefix+"_gc", func() {
		runtime.GC()
		runtime.GC()
	})
	runtime.ReadMemStats(&m1)
	body()
	runtime.ReadMemStats(&m2)
	return float64(m2.Alloc-m1.Alloc) / 1024 / 1024
}

func createTestTrie(n, capacity, offset, stride int) *IntervalSet {
	r := NewIntervalSetWithCapacity(uint(capacity), Above(0))
	var s IntervalSet
	for i := 0; i < n; i++ {
		s = Above(uint64(i*stride + offset)).AsIntervalSet()
		r = r.SymmetricDifference(r, &s)
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
		n int = 30000
		m int = 30000
		t int = 5
	)

	// Disable GC so we get better measurements
	debug.SetGCPercent(-1)

	Measure("or", func(bm Benchmarker) {
		var a, b *IntervalSet
		bm.Time("total", func() {
			alloc := measureMemoryUsageDuringOperation(bm, "creation", func() {
				creation := bm.Time("creation", func() {
					a = createTestTrie(n, m, 0, 2)
					b = createTestTrie(n, m, 1, 2)
				})
				Ω(creation.Seconds()).Should(BeNumerically("<", 15))
			})
			bm.RecordValue("creationAlloc", alloc)
			alloc = measureMemoryUsageDuringOperation(bm, "runtime", func() {
				runtime := bm.Time("runtime", func() {
					a.Union(a, b)
				})
				Ω(runtime.Seconds()).Should(BeNumerically("<", 2))
			})
			bm.RecordValue("execAlloc", alloc)
		})
	}, t)

	Measure("or - old", func(bm Benchmarker) {
		var a, b *temporalset.IntervalSet
		bm.Time("total", func() {
			alloc := measureMemoryUsageDuringOperation(bm, "creation", func() {
				bm.Time("creation", func() {
					a = createOldTestTrie(n, m, 0, 2)
					b = createOldTestTrie(n, m, 1, 2)
				})
			})
			bm.RecordValue("creationAlloc", alloc)
			alloc = measureMemoryUsageDuringOperation(bm, "runtime", func() {
				runtime := bm.Time("runtime", func() {
					a.Union(b)
				})
				Ω(runtime.Seconds()).Should(BeNumerically("<", 2))
			})
			bm.RecordValue("execAlloc", alloc)
		})
	}, t)

	Measure("and", func(bm Benchmarker) {
		var a, b *IntervalSet
		bm.Time("total", func() {
			alloc := measureMemoryUsageDuringOperation(bm, "creation", func() {
				creation := bm.Time("creation", func() {
					a = createTestTrie(n, m, 0, 2)
					b = createTestTrie(n, m, 1, 2)
				})
				Ω(creation.Seconds()).Should(BeNumerically("<", 15))
			})
			bm.RecordValue("creationAlloc", alloc)
			alloc = measureMemoryUsageDuringOperation(bm, "runtime", func() {
				runtime := bm.Time("runtime", func() {
					a.Intersection(a, b)
				})
				Ω(runtime.Seconds()).Should(BeNumerically("<", 2))
			})
			bm.RecordValue("execAlloc", alloc)
		})

	}, t)

	Measure("and - old", func(bm Benchmarker) {
		var a, b *temporalset.IntervalSet
		bm.Time("total", func() {
			alloc := measureMemoryUsageDuringOperation(bm, "creation", func() {
				bm.Time("creation", func() {
					a = createOldTestTrie(n, m, 0, 2)
					b = createOldTestTrie(n, m, 1, 2)
				})
			})
			bm.RecordValue("creationAlloc", alloc)
			alloc = measureMemoryUsageDuringOperation(bm, "runtime", func() {
				runtime := bm.Time("runtime", func() {
					a.Intersection(b)
				})
				Ω(runtime.Seconds()).Should(BeNumerically("<", 2))
			})
			bm.RecordValue("execAlloc", alloc)
		})
	}, t)

})

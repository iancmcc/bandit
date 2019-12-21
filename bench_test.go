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

func createTestTrie(n, capacity, offset, stride int) *IntervalSet {
	r := NewIntervalSetWithCapacity(uint(capacity), Above(0))
	for i := 0; i < n; i++ {
		r = r.SymmetricDifference(r, NewIntervalSetWithCapacity(4, Above(uint64(i*stride+offset))))
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
		n int = 200
		m int = 6000
	)

	Measure("or", func(bm Benchmarker) {
		var a, b *IntervalSet
		alloc := measureMemoryUsageDuringOperation(func() {
			creation := bm.Time("creation", func() {
				a = createTestTrie(n, m, 0, 2)
				b = createTestTrie(n, m, 1, 2)
			})
			Ω(creation.Seconds()).Should(BeNumerically("<", 15))
		})
		bm.RecordValue("creationAlloc", alloc)
		alloc = measureMemoryUsageDuringOperation(func() {
			runtime := bm.Time("runtime", func() {
				a.Union(a, b)
			})
			Ω(runtime.Seconds()).Should(BeNumerically("<", 2))
		})
		bm.RecordValue("execAlloc", alloc)
	}, 1000)

	Measure("or - old", func(bm Benchmarker) {
		var a, b *temporalset.IntervalSet
		alloc := measureMemoryUsageDuringOperation(func() {
			bm.Time("creation", func() {
				a = createOldTestTrie(n, m, 0, 2)
				b = createOldTestTrie(n, m, 1, 2)
			})
		})
		bm.RecordValue("creationAlloc", alloc)
		alloc = measureMemoryUsageDuringOperation(func() {
			runtime := bm.Time("runtime", func() {
				a.Union(b)
			})
			Ω(runtime.Seconds()).Should(BeNumerically("<", 2))
		})
		bm.RecordValue("execAlloc", alloc)

	}, 1000)

	Measure("and", func(bm Benchmarker) {
		var a, b *IntervalSet
		alloc := measureMemoryUsageDuringOperation(func() {
			creation := bm.Time("creation", func() {
				a = createTestTrie(n, m, 0, 2)
				b = createTestTrie(n, m, 1, 2)
			})
			Ω(creation.Seconds()).Should(BeNumerically("<", 15))
		})
		bm.RecordValue("creationAlloc", alloc)
		alloc = measureMemoryUsageDuringOperation(func() {
			runtime := bm.Time("runtime", func() {
				a.Intersection(a, b)
			})
			Ω(runtime.Seconds()).Should(BeNumerically("<", 2))
		})
		bm.RecordValue("execAlloc", alloc)

	}, 1000)

	Measure("and - old", func(bm Benchmarker) {
		var a, b *temporalset.IntervalSet
		alloc := measureMemoryUsageDuringOperation(func() {
			bm.Time("creation", func() {
				a = createOldTestTrie(n, m, 0, 2)
				b = createOldTestTrie(n, m, 1, 2)
			})
		})
		bm.RecordValue("creationAlloc", alloc)
		alloc = measureMemoryUsageDuringOperation(func() {
			runtime := bm.Time("runtime", func() {
				a.Intersection(b)
			})
			Ω(runtime.Seconds()).Should(BeNumerically("<", 2))
		})
		bm.RecordValue("execAlloc", alloc)
	}, 1000)

})

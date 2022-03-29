package bandit_test

import (
	"math"
	"testing"

	"github.com/b97tsk/rangeset"
	"github.com/iancmcc/bandit"
)

const (
	n      = 1000
	m      = 1000
	offset = 0
	stride = 2
)

func BenchmarkRangeSet(b *testing.B) {

	var (
		r rangeset.RangeSet
		q rangeset.RangeSet
	)
	b.Run("creating", func(b *testing.B) {
		b.ReportAllocs()
		for j := 0; j < b.N; j++ {
			r = make(rangeset.RangeSet, m)
			q = make(rangeset.RangeSet, m)
			for i := int64(0); i < n; i++ {
				s := rangeset.FromRange(int64(i*stride+offset), math.MaxInt64)
				t := rangeset.FromRange(int64(i*stride+offset+1), math.MaxInt64)
				r = rangeset.SymmetricDifference(r, s)
				q = rangeset.SymmetricDifference(q, t)
			}
		}
		b.Run("intersection", func(b *testing.B) {
			b.ReportAllocs()
			for j := 0; j < b.N; j++ {
				r.Intersection(q)
			}
		})
		b.Run("union", func(b *testing.B) {
			b.ReportAllocs()
			for j := 0; j < b.N; j++ {
				r.Union(q)
			}
		})
	})
}

func BenchmarkBanditIntervalSet(b *testing.B) {

	var (
		r *bandit.IntervalSet
		q *bandit.IntervalSet
	)

	b.Run("creating", func(b *testing.B) {
		b.ReportAllocs()
		for j := 0; j < b.N; j++ {
			r = bandit.NewIntervalSetWithCapacity(uint(m), bandit.Above(0))
			q = bandit.NewIntervalSetWithCapacity(uint(m), bandit.Above(0))
			for i := 0; i < n; i++ {
				s := bandit.Above(uint64(i*stride + offset)).AsIntervalSet()
				t := bandit.Above(uint64(i*stride + offset + 1)).AsIntervalSet()
				r = r.SymmetricDifference(r, s)
				q = q.SymmetricDifference(q, t)
			}
		}
		b.Run("intersection", func(b *testing.B) {
			b.ReportAllocs()
			for j := 0; j < b.N; j++ {
				bandit.NewIntervalSetWithCapacity(uint(m)).Intersection(r, q)
			}
		})
		b.Run("union", func(b *testing.B) {
			b.ReportAllocs()
			for j := 0; j < b.N; j++ {
				bandit.NewIntervalSetWithCapacity(uint(m)).Union(r, q)
			}
		})
	})

}

package bandit

import (
	"fmt"
	"strings"

	"golang.org/x/exp/rand"
)

type (
	setnode struct {
		IntervalSet
		ptr uint
	}
	IntervalMap struct {
		m        map[interface{}]uint // TODO: Make this more GC-friendly
		ncap     uint
		numfree  uint
		nextfree uint
		sets     []setnode
	}
)

const defaultMapCapacity = 32

func NewIntervalMap() *IntervalMap {
	return NewIntervalMapWithCapacity(defaultMapCapacity, defaultIntervalSetCapacity)
}

func NewIntervalMapWithCapacity(mapCapacity, nodeArraySize int) *IntervalMap {
	var m IntervalMap
	m.m = make(map[interface{}]uint, mapCapacity)
	m.ncap = uint(nodeArraySize)
	m.sets = make([]setnode, 1, mapCapacity)
	return &m
}

func CopyMap(x *IntervalMap) *IntervalMap {
	return NewMap(x.Caps()).Copy(x)
}

func NewMap(mc, nc int) *IntervalMap {
	return NewIntervalMapWithCapacity(mc, nc)
}

func (z *IntervalMap) String() string {
	b := new(strings.Builder)
	for k, iset := range z.m {
		fmt.Fprintf(b, "%s: %s\n", k, &z.sets[iset].IntervalSet)
	}
	return b.String()
}

func (z *IntervalMap) Caps() (int, int) {
	return cap(z.sets), int(z.ncap)
}

func (z *IntervalMap) IsEmpty() bool {
	return len(z.m) == 0
}

func (z *IntervalMap) Equals(x *IntervalMap) bool {
	if z == x {
		return true
	}
	if x == nil {
		return false
	}
	if len(z.m) != len(x.m) {
		return false
	}
	for k, zidx := range z.m {
		xidx, ok := x.m[k]
		if !ok {
			return false
		}
		if !(&z.sets[zidx].IntervalSet).Equals(&x.sets[xidx].IntervalSet) {
			return false
		}
	}
	return true
}

func CopySet(x *IntervalSet) *IntervalSet {
	return NewSet(x.Cap()).Copy(x)
}

func (z *IntervalMap) allocset(x *IntervalSet, maxcap uint) (idx uint) {
	if z.numfree > 0 {
		idx = z.nextfree
		s := &z.sets[idx]
		z.nextfree, s.ptr = s.ptr, 0
		s.IntervalSet.Copy(x)
		z.numfree -= 1
	} else {
		if x == nil {
			if z.ncap > maxcap {
				maxcap = z.ncap
			}
			x = NewIntervalSetWithCapacity(maxcap)
		} else {
			x = CopySet(x)
		}
		z.sets = append(z.sets, setnode{*x, 0})
		idx = uint(len(z.sets) - 1)
	}
	return
}

func (z *IntervalMap) free(idx uint) {
	if idx == 0 {
		return
	}
	z.sets[idx] = setnode{ptr: z.nextfree}
	z.nextfree = idx
	z.numfree += 1
}

func (z *IntervalMap) Clear() {
	z.sets = z.sets[:1]
	z.nextfree = 0
	z.numfree = 0
	// A compiler optimization makes this fast
	for k := range z.m {
		delete(z.m, k)
	}
}

func (z *IntervalMap) Copy(x *IntervalMap) *IntervalMap {
	if z == x {
		return x
	}
	z.Clear()
	if x == nil {
		return z
	}
	for k, v := range x.m {
		z.m[k] = z.allocset(&x.sets[v].IntervalSet, 0)
	}
	return z
}

func (z *IntervalMap) Cardinality() int {
	return len(z.m)
}

func (z *IntervalMap) Add(x *IntervalMap, val interface{}, ival ...Interval) *IntervalMap {
	switch {
	case x == nil:
		z.Clear()
	case z != x:
		z.Copy(x)
	}
	idx, ok := z.m[val]
	if !ok {
		idx = z.allocset(nil, uint(len(ival)))
		z.m[val] = idx
	}
	set := &z.sets[idx].IntervalSet
	set.Add(set, ival...)
	if set.IsEmpty() {
		z.remove(val)
	}
	return z
}

func (z *IntervalMap) AddSet(x *IntervalMap, val interface{}, iset *IntervalSet) *IntervalMap {
	switch {
	case x == nil:
		z.Clear()
	case z != x:
		z.Copy(x)
	}
	if iset == nil || iset.IsEmpty() {
		return z
	}
	idx, ok := z.m[val]
	if !ok {
		idx = z.allocset(iset, 0)
		z.m[val] = idx
		return z
	}
	set := &z.sets[idx].IntervalSet
	set.Union(set, iset)
	if set.IsEmpty() {
		z.remove(val)
	}
	return z
}

func (z *IntervalMap) Intersection(x *IntervalMap, y *IntervalMap) *IntervalMap {
	if x == nil || y == nil {
		z.Clear()
		return z
	}
	if x == y {
		z.Copy(x)
		return z
	}
	if z == x || z == y {
		var other *IntervalMap
		if z == x {
			other = y
		} else {
			other = x
		}
		for k, zidx := range z.m {
			if oidx, ok := other.m[k]; !ok {
				z.remove(k)
			} else {
				zset, oset := &z.sets[zidx].IntervalSet, &other.sets[oidx].IntervalSet
				if zset.Intersection(zset, oset).IsEmpty() {
					z.remove(k)
				}
			}
		}
		return z
	}
	z.Clear()
	var (
		xn, yn           = x.Cardinality(), y.Cardinality()
		aidx, bidx, zidx uint
		smaller, larger  *IntervalMap
		aset, bset, zset *IntervalSet
		ok               bool
		k                interface{}
	)
	if xn < yn {
		smaller, larger = x, y
	} else {
		smaller, larger = y, x
	}
	for k, aidx = range smaller.m {
		if bidx, ok = larger.m[k]; !ok {
			continue
		}
		aset, bset = &smaller.sets[aidx].IntervalSet, &larger.sets[bidx].IntervalSet
		zidx = z.allocset(nil, 0)
		zset = &z.sets[zidx].IntervalSet
		if zset.Intersection(aset, bset).IsEmpty() {
			z.remove(zidx)
			continue
		}
		z.m[k] = zidx
	}
	return z
}

func (z *IntervalMap) Union(x *IntervalMap, y *IntervalMap) *IntervalMap {
	switch {
	case x == nil, x == y:
		z.Copy(y)
	case y == nil:
		z.Copy(x)
	case z != x && z != y:
		z.Copy(x)
		x = z
		fallthrough
	case z == x || z == y:
		var other *IntervalMap
		if z == x {
			other = y
		} else {
			other = x
		}
		for k, oidx := range other.m {
			oset := &other.sets[oidx].IntervalSet
			zidx, ok := z.m[k]
			if !ok {
				z.m[k] = z.allocset(oset, 0)
				continue
			}
			zset := &z.sets[zidx].IntervalSet
			zset.Union(zset, oset)
		}
	}
	return z
}

func (z *IntervalMap) remove(k interface{}) bool {
	idx, ok := z.m[k]
	if !ok {
		return false
	}
	z.free(idx)
	delete(z.m, k)
	return true
}

func (z *IntervalMap) SymmetricDifference(x *IntervalMap, y *IntervalMap) *IntervalMap {
	switch {
	case x == nil:
		z.Copy(y)
	case y == nil:
		z.Copy(x)
	case x == y:
		z.Clear()
	case z != x && z != y:
		z.Copy(x)
		x = z
		fallthrough
	default:
		var other *IntervalMap
		if z == x {
			other = y
		} else {
			other = x
		}
		for k, oidx := range other.m {
			oset := &other.sets[oidx].IntervalSet
			zidx, ok := z.m[k]
			if !ok {
				z.m[k] = z.allocset(oset, 0)
				continue
			}
			zset := &z.sets[zidx].IntervalSet
			zset.SymmetricDifference(zset, oset)
			if zset.IsEmpty() {
				z.remove(k)
			}
		}
	}
	return z
}

func (z *IntervalMap) Check() {
	// Check free list
	// Check free list
	if z.nextfree > 0 {
		n := z.nextfree
		c := z.numfree
		for n > 0 {
			nd := &z.sets[n]
			n = nd.ptr
			c -= 1
		}
		if c != 0 {
			fmt.Println("ERROR: Free list was incorrect")
		}
	}

	// Check sets
	for k, idx := range z.m {
		func(k interface{}, idx uint) {
			defer func() {
				if e := recover(); e != nil {
					fmt.Println("OFFENDING KEY: ", k)
					panic(e)
				}
			}()
			set := &z.sets[idx].IntervalSet
			if set.IsEmpty() {
				panic("INTERVALSET IS EMPTY")
			}
			(&z.sets[idx].IntervalSet).Check()
		}(k, idx)
	}
}

func (z *IntervalMap) AllIntervals() *IntervalSet {
	s := NewIntervalSetWithCapacity(z.ncap)
	for k, sidx := range z.m {
		set := &z.sets[sidx].IntervalSet
		check(set, fmt.Sprintf("SET %s", k))
		s.Union(s, set)
		check(s, fmt.Sprintf("S POST UNION WITH %s", k))
	}
	check(s, "S ALL INTERVALS")
	return s
}

func (z *IntervalMap) MutateValues(x *IntervalMap, f func(interface{}) interface{}) *IntervalMap {
	z.Copy(x)
	seen := make(map[interface{}]struct{})
	for k, idx := range z.m {
		if _, ok := seen[k]; ok {
			continue
		}
		nv := f(k)
		if nv == nil {
			// We won't include this key
			z.remove(k)
			continue
		}
		if existing, ok := z.m[nv]; ok {
			// Merge existing
			x1 := &z.sets[existing].IntervalSet
			x1.Union(x1, &z.sets[idx].IntervalSet)
			if k != nv {
				z.remove(k)
			}
			continue
		}
		if nv != k {
			z.m[nv] = idx
			seen[nv] = struct{}{}
			delete(z.m, k)
		}
	}
	return z
}

func (z *IntervalMap) Mask(x *IntervalMap, mask *IntervalSet) *IntervalMap {
	if z != x {
		z.Clear()
	}
	for k, idx := range x.m {
		s := &x.sets[idx].IntervalSet
		if z == x {
			s.Intersection(s, mask)
			continue
		}
		didx := z.allocset(nil, 0)
		if (&z.sets[didx].IntervalSet).Intersection(s, mask).IsEmpty() {
			z.free(didx)
			continue
		}
		z.m[k] = didx
	}
	return z
}

func (z *IntervalMap) MaskEnclosed(x *IntervalMap, mask *IntervalSet) *IntervalMap {
	if z != x {
		z.Clear()
	}
	for k, idx := range x.m {
		s := &x.sets[idx].IntervalSet
		if z == x {
			if s.Enclosed(mask, s).IsEmpty() {
				z.remove(k)
			}
			continue
		}
		didx := z.allocset(nil, 0)
		if (&z.sets[didx].IntervalSet).Enclosed(mask, s).IsEmpty() {
			z.free(didx)
			continue
		}
		z.m[k] = didx
	}
	return z
}

func (z *IntervalMap) SubtractInterval(x *IntervalMap, value interface{}, ival Interval) *IntervalMap {
	if z != x {
		z.Copy(x)
	}
	idx, ok := z.m[value]
	if !ok {
		return z
	}
	set := &z.sets[idx].IntervalSet
	if set.Difference(set, ival.AsIntervalSet()).IsEmpty() {
		z.remove(value)
	}
	return z
}

func (z *IntervalMap) PopMask(x *IntervalMap, set *IntervalSet) (*IntervalMap, *IntervalMap) {
	popped := NewMap(x.Caps()).Mask(x, set)
	if z != x {
		z.Clear()
	}
	z.Difference(x, popped)
	return z, popped
}

func (z *IntervalMap) ValueSlice() []interface{} {
	s := make([]interface{}, 0, len(z.m)+1)
	for k := range z.m {
		s = append(s, k)
	}
	return s
}

func (z *IntervalMap) Difference(x *IntervalMap, y *IntervalMap) *IntervalMap {
	switch {
	case y == nil:
		z.Copy(x)
	case x == nil, x == y:
		z.Clear()
	case z != x && z != y:
		z.Copy(x)
		x = z
		fallthrough
	case z == x:
		for k, zidx := range z.m {
			yidx, ok := y.m[k]
			if !ok {
				continue
			}
			zset := &z.sets[zidx].IntervalSet
			yset := &y.sets[yidx].IntervalSet
			if zset.Difference(zset, yset).IsEmpty() {
				z.remove(k)
			}
		}
	case z == y:
		fmt.Println("2")
		for k, xidx := range x.m {
			xset := &x.sets[xidx].IntervalSet
			zidx, ok := z.m[k]
			if !ok {
				z.m[k] = z.allocset(xset, 0)
				continue
			}
			zset := &z.sets[zidx].IntervalSet
			if zset.Difference(xset, zset).IsEmpty() {
				z.remove(k)
			}
		}
	}
	return z
}

func (z *IntervalMap) Iterator() *MapIterator {
	return NewMapIterator(z)
}

func (z *IntervalMap) PopRandValue(x *IntervalMap, rng *rand.Rand, alpha float64) (*IntervalMap, interface{}, Interval) {
	z.Copy(x)
	k, ival := z.RandValue(rng, alpha)
	if !ival.IsEmpty() {
		s := &z.sets[z.m[k]].IntervalSet
		if s.Difference(s, ival.AsIntervalSet()).IsEmpty() {
			z.remove(k)
		}
	}
	return z, k, ival
}

func (z *IntervalMap) RandValue(rng *rand.Rand, alpha float64) (interface{}, Interval) {
	if rng.Float64() > alpha {
		return nil, Empty()
	}
	var total int
	for _, set := range z.sets {
		if set.IsEmpty() {
			continue
		}
		if set.Cardinality() == 0 {
			total += 1
			continue
		}
		total += set.Cardinality()
	}
	if total == 0 {
		return nil, Empty()
	}
	target := rng.Intn(total)
	for k, setidx := range z.m {
		set := &z.sets[setidx]
		target -= set.Cardinality()
		if target <= -1 {
			return k, set.RandInterval(rng)
		}
	}
	return nil, Empty()
}

func (z *IntervalMap) Enclosed(x *IntervalMap, y *IntervalMap) *IntervalMap {
	switch {
	case x == y:
		z.Copy(x)
	case x == nil, y == nil:
		z.Clear()
	case z != x && z != y:
		z.Copy(x)
		x = z
		fallthrough
	case z == x:
		for k, zidx := range z.m {
			yidx, ok := y.m[k]
			if !ok {
				z.remove(k)
				continue
			}
			zset := &z.sets[zidx].IntervalSet
			yset := &y.sets[yidx].IntervalSet
			if zset.Enclosed(zset, yset).IsEmpty() {
				z.remove(k)
			}
		}
	case z == y:
		for k, xidx := range x.m {
			xset := &x.sets[xidx].IntervalSet
			zidx, ok := z.m[k]
			if !ok {
				z.m[k] = z.allocset(xset, 0)
				continue
			}
			zset := &z.sets[zidx].IntervalSet
			if zset.Enclosed(xset, zset).IsEmpty() {
				z.remove(k)
			}
		}
	}
	return z
}

func (z *IntervalMap) Get(value interface{}) *IntervalSet {
	idx, ok := z.m[value]
	if !ok {
		return NewIntervalSetWithCapacity(1)
	}
	return NewIntervalSet().Copy(&z.sets[idx].IntervalSet)
}

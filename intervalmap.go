package bandit

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

func NewIntervalMapWithCapacity(mapCapacity, nodeArraySize uint) *IntervalMap {
	var m IntervalMap
	m.m = make(map[interface{}]uint, mapCapacity)
	m.ncap = nodeArraySize
	m.sets = make([]setnode, 1, mapCapacity)
	return &m
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
	for k, v := range x.m {
		z.m[k] = v
	}
	z.sets = append(z.sets[:1], x.sets[:1]...)
	z.nextfree = x.nextfree
	z.numfree = x.numfree
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
			z.free(zidx)
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

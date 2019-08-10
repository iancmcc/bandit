package bandit

import "math"

const (
	vTypeUint64 vtype = iota
	vTypeInt64
	vTypeFloat64
)

func i64(v int64) uint64 {
	return uint64(v) ^ (1 << 63)
}

func f64(v float64) uint64 {
	tv := int64(math.Float64bits(v))
	tv ^= (tv >> 63) | (-1 << 63)
	return uint64(tv)
}

func toUint64(v interface{}) (uint64, vtype, error) {
	switch v := v.(type) {
	case int8:
		return i64(int64(v)), vTypeInt64, nil
	case int16:
		return i64(int64(v)), vTypeInt64, nil
	case int:
		return i64(int64(v)), vTypeInt64, nil
	case int32:
		return i64(int64(v)), vTypeInt64, nil
	case int64:
		return i64(v), vTypeInt64, nil
	case uint8:
		return uint64(v), vTypeUint64, nil
	case uint16:
		return uint64(v), vTypeUint64, nil
	case uint:
		return uint64(v), vTypeUint64, nil
	case uint32:
		return uint64(v), vTypeUint64, nil
	case uint64:
		return v, vTypeUint64, nil
	case float32:
		return f64(float64(v)), vTypeFloat64, nil
	case float64:
		return f64(v), vTypeFloat64, nil
	}
	return 0, 0, ErrInvalidInterval
}

func fromUint64(v uint64, t vtype) interface{} {
	switch t {
	case vTypeUint64:
		return v
	case vTypeInt64:
		return int64(v ^ (1 << 63))
	case vTypeFloat64:
		i := int64(v)
		i ^= (^i >> 63) | (-1 << 63)
		return math.Float64frombits(uint64(i))
	}
	panic("never")
}

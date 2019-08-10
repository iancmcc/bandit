package bandit

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	empty   = "(Ø)"
	inf     = "∞"
	ninf    = "-∞"
	lclosed = "["
	rclosed = "]"
	lopen   = "("
	ropen   = ")"
	delim   = ", "
)

var (
	pat                = regexp.MustCompile(`^([\(\[])([^,]+), ?([^\]\)]+)([\]\)])$`)
	ErrInvalidInterval = errors.New("invalid interval")
)

func (ival Interval) String() string {
	if ival.IsEmpty() {
		return empty
	}
	var sb strings.Builder
	if ival.leftUnbounded {
		sb.WriteString(lopen)
		sb.WriteString(ninf)
	} else {
		if ival.leftInclusive {
			sb.WriteString(lclosed)
		} else {
			sb.WriteString(lopen)
		}
		sb.WriteString(fmt.Sprintf("%v", fromUint64(ival.lower, ival.vtype)))
	}
	sb.WriteString(delim)
	if ival.rightUnbounded {
		sb.WriteString(inf)
		sb.WriteString(ropen)
	} else {
		sb.WriteString(fmt.Sprintf("%v", fromUint64(ival.upper, ival.vtype)))
		if ival.rightInclusive {
			sb.WriteString(rclosed)
		} else {
			sb.WriteString(ropen)
		}
	}
	return sb.String()
}

func parse(s string) (li, ri, lu, ru bool, l, u string, err error) {
	groups := pat.FindStringSubmatch(s)
	if len(groups) == 0 {
		err = ErrInvalidInterval
		return
	}
	li = groups[1] == lclosed
	ri = groups[4] == rclosed
	lu = groups[2] == ninf
	ru = groups[3] == inf
	if !lu {
		l = groups[2]
	}
	if !ru {
		u = groups[3]
	}
	return
}

func Uint(s string) (ival Interval, err error) {
	var l, u string
	ival.leftInclusive, ival.rightInclusive, ival.leftUnbounded, ival.rightUnbounded, l, u, err = parse(s)
	if err != nil {
		return
	}
	if l != "" {
		if ival.lower, err = strconv.ParseUint(l, 10, 64); err != nil {
			return
		}
	}
	if u != "" {
		if ival.upper, err = strconv.ParseUint(u, 10, 64); err != nil {
			return
		}
	}
	ival.vtype = vTypeUint64
	return
}

func Int(s string) (ival Interval, err error) {
	var l, u string
	ival.leftInclusive, ival.rightInclusive, ival.leftUnbounded, ival.rightUnbounded, l, u, err = parse(s)
	if err != nil {
		return
	}
	var il, iu int64
	if l != "" {
		if il, err = strconv.ParseInt(l, 10, 64); err != nil {
			return
		}
		if ival.lower, _, err = toUint64(il); err != nil {
			return
		}
	}
	if u != "" {
		if iu, err = strconv.ParseInt(u, 10, 64); err != nil {
			return
		}
		if ival.upper, _, err = toUint64(iu); err != nil {
			return
		}
	}
	ival.vtype = vTypeInt64
	return
}

func Float(s string) (ival Interval, err error) {
	var l, u string
	ival.leftInclusive, ival.rightInclusive, ival.leftUnbounded, ival.rightUnbounded, l, u, err = parse(s)
	if err != nil {
		return
	}
	var il, iu float64
	if l != "" {
		if il, err = strconv.ParseFloat(l, 64); err != nil {
			return
		}
		if ival.lower, _, err = toUint64(il); err != nil {
			return
		}
	}
	if u != "" {
		if iu, err = strconv.ParseFloat(u, 64); err != nil {
			return
		}
		if ival.upper, _, err = toUint64(iu); err != nil {
			return
		}
	}
	ival.vtype = vTypeFloat64
	return
}

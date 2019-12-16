package bandit

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"
)

const (
	stageLowerBound = iota
	stageLower
	stageUpper
	stageUpperBound
)

func ParseInterval(b []byte) (Interval, error) {
	return parse(bytes.NewReader(b))
}

func ParseIntervalString(s string) (Interval, error) {
	return parse(strings.NewReader(s))
}

func MustParseInterval(b []byte) Interval {
	ival, err := ParseInterval(b)
	if err != nil {
		panic(err)
	}
	return ival
}

func MustParseIntervalString(s string) Interval {
	ival, err := ParseIntervalString(s)
	if err != nil {
		panic(err)
	}
	return ival
}

func isIdent(ch rune, i int) bool {
	return ch == '-' && i == 0 || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
}

func parse(src io.Reader) (ival Interval, err error) {
	var s scanner.Scanner
	s.Init(src)
	s.IsIdentRune = isIdent

	stage := stageLowerBound

	var lowerBound, upperBound BoundType
	var lower, upper uint64

out:
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		switch stage {
		case stageLowerBound:
			switch tok {
			case '(':
				lowerBound = OpenBound
			case '[':
				lowerBound = ClosedBound
			default:
				err = ErrInvalidInterval
				return
			}
			stage = stageLower
		case stageLower:
			t := s.TokenText()
			switch t {
			case "-∞", "-inf", "-Inf":
				lowerBound = UnboundBound
			default:
				l, perr := strconv.ParseUint(t, 10, 64)
				if perr != nil {
					err = ErrInvalidInterval
					return
				}
				lower = l
			}
			stage = stageUpper
		case stageUpper:
			t := s.TokenText()
			switch t {
			case ",":
				continue
			case "∞", "inf", "Inf":
				upperBound = UnboundBound
				break out
			default:
				u, perr := strconv.ParseUint(t, 10, 64)
				if perr != nil {
					err = ErrInvalidInterval
					return
				}
				upper = u
				stage = stageUpperBound
			}
		case stageUpperBound:
			switch tok {
			case ')':
				upperBound = OpenBound
			case ']':
				upperBound = ClosedBound
			default:
				err = ErrInvalidInterval
				return
			}
		}
	}
	return NewInterval(lowerBound, lower, upper, upperBound), nil
}

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

func (ival *Interval) ParseInterval(b []byte) error {
	return ival.parse(bytes.NewReader(b))
}

func (ival *Interval) ParseIntervalString(s string) error {
	return ival.parse(strings.NewReader(s))
}

func (ival *Interval) parse(src io.Reader) error {
	var s scanner.Scanner
	s.Init(src)
	s.IsIdentRune = func(ch rune, i int) bool {
		return ch == '-' && i == 0 || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
	}

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
				return ErrInvalidInterval
			}
			stage = stageLower
		case stageLower:
			t := s.TokenText()
			switch t {
			case "-∞", "-inf", "-Inf":
				lowerBound = UnboundBound
			default:
				l, err := strconv.ParseUint(t, 10, 64)
				if err != nil {
					return ErrInvalidInterval
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
				u, err := strconv.ParseUint(t, 10, 64)
				if err != nil {
					return ErrInvalidInterval
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
				return ErrInvalidInterval
			}
		}
	}
	ival.SetInterval(lowerBound, lower, upper, upperBound)
	return nil
}

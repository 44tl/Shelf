package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func ParseDuration(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || s == "0" {
		return 0, nil
	}
	var total time.Duration
	numStart := -1
	for i, r := range s {
		switch {
		case (r >= '0' && r <= '9') || r == '.':
			if numStart < 0 {
				numStart = i
			}
		case r == 's' || r == 'm' || r == 'h' || r == 'd' || r == 'w':
			if numStart < 0 {
				return 0, fmt.Errorf("shelf: bad duration %q: unit %q has no number", s, r)
			}
			n, err := strconv.ParseFloat(s[numStart:i], 64)
			if err != nil {
				return 0, fmt.Errorf("shelf: bad duration %q: %v", s, err)
			}
			var unit time.Duration
			switch r {
			case 's':
				unit = time.Second
			case 'm':
				unit = time.Minute
			case 'h':
				unit = time.Hour
			case 'd':
				unit = 24 * time.Hour
			case 'w':
				unit = 7 * 24 * time.Hour
			}
			ns := n * float64(unit)
			if !(ns >= 0) || ns > math.MaxInt64 {
				return 0, fmt.Errorf("shelf: duration %q is out of range", s)
			}
			total += time.Duration(ns)
			numStart = -1
		default:
			return 0, fmt.Errorf("shelf: unexpected character %q in duration %q", string(r), s)
		}
		if total < 0 {
			return 0, fmt.Errorf("shelf: duration %q is out of range", s)
		}
	}
	if numStart >= 0 {
		return 0, fmt.Errorf("shelf: bad duration %q: missing time unit", s)
	}
	return total, nil
}

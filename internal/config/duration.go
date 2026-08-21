package config

import (
	"fmt"
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
			switch r {
			case 's':
				total += time.Duration(n * float64(time.Second))
			case 'm':
				total += time.Duration(n * float64(time.Minute))
			case 'h':
				total += time.Duration(n * float64(time.Hour))
			case 'd':
				total += time.Duration(n * 24 * float64(time.Hour))
			case 'w':
				total += time.Duration(n * 7 * 24 * float64(time.Hour))
			}
			numStart = -1
		default:
			return 0, fmt.Errorf("shelf: unexpected character %q in duration %q", string(r), s)
		}
	}
	if numStart >= 0 {
		return 0, fmt.Errorf("shelf: bad duration %q: missing time unit", s)
	}
	return total, nil
}

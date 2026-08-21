package config

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		err  bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"45s", 45 * time.Second, false},
		{"90m", 90 * time.Minute, false},
		{"24h", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"1w2d", 9 * 24 * time.Hour, false},
		{"1.5d", 36 * time.Hour, false},
		{"7D", 7 * 24 * time.Hour, false},
		{"d", 0, true},
		{"7", 0, true},
		{"soon", 0, true},
		{"7x", 0, true},
	}
	for _, c := range cases {
		got, err := ParseDuration(c.in)
		if c.err {
			if err == nil {
				t.Errorf("ParseDuration(%q): expected error, got %v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDuration(%q): unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseDuration(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

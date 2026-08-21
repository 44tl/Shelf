package ui

import (
	"strings"
	"testing"
	"time"
)

func TestWidthsClampLongValues(t *testing.T) {
	nameW, destW := Widths(
		[]string{"short.txt", strings.Repeat("a", 100)},
		[]string{"~/x", "~/" + strings.Repeat("b", 100)},
	)
	if nameW != 44 {
		t.Errorf("nameW = %d, want 44", nameW)
	}
	if destW != 40 {
		t.Errorf("destW = %d, want 40", destW)
	}
}

func TestWidthsKeepSensibleMinimums(t *testing.T) {
	nameW, destW := Widths([]string{"a"}, []string{"b"})
	if nameW != 16 || destW != 28 {
		t.Errorf("got %d/%d, want minimums 16/28", nameW, destW)
	}
}

func TestAgeClampsFutureTimestamps(t *testing.T) {
	if got := Age(-2 * time.Hour); got != "0h" {
		t.Errorf("Age(future) = %q, want \"0h\"", got)
	}
	if got := Age(3 * 24 * time.Hour); got != "3d" {
		t.Errorf("Age(3d) = %q, want \"3d\"", got)
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:            "0 B",
		-5:           "0 B",
		512:          "512 B",
		2 << 10:      "2.0 KB",
		5 << 20:      "5.0 MB",
		int64(1.5e9): "1.4 GB",
	}
	for in, want := range cases {
		if got := HumanBytes(in); got != want {
			t.Errorf("HumanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}

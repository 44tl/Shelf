package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	Bold   = "\x1b[1m"
	Dim    = "\x1b[2m"
	Green  = "\x1b[32m"
	Yellow = "\x1b[33m"
	Cyan   = "\x1b[36m"
	Red    = "\x1b[31m"
	Reset  = "\x1b[0m"
)

func Init() {
	if os.Getenv("NO_COLOR") != "" {
		disable()
		return
	}
	fi, err := os.Stdout.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		disable()
	}
}

func disable() {
	Bold, Dim, Green, Yellow, Cyan, Red, Reset = "", "", "", "", "", "", ""
}

var Home string

func Shorten(p string) string {
	if Home == "" {
		return p
	}
	if p == Home {
		return "~"
	}
	if strings.HasPrefix(p, Home+string(filepath.Separator)) {
		return "~" + p[len(Home):]
	}
	return p
}

func Trunc(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func RuneWidth(s string) int { return len([]rune(s)) }

func HumanBytes(n int64) string {
	const kb, mb, gb, tb = 1 << 10, 1 << 20, 1 << 30, 1 << 40
	switch {
	case n >= tb:
		return fmt.Sprintf("%.1f TB", float64(n)/tb)
	case n >= gb:
		return fmt.Sprintf("%.1f GB", float64(n)/gb)
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/mb)
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/kb)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func Age(d time.Duration) string {
	h := int(d.Hours())
	if h < 48 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dd", h/24)
}

func Widths(names, dests []string) (nameW, destW int) {
	nameW, destW = 16, 28
	for _, s := range names {
		if w := RuneWidth(s); w > nameW && w <= 44 {
			nameW = w
		}
	}
	for _, s := range dests {
		if w := RuneWidth(s); w > destW && w <= 40 {
			destW = w
		}
	}
	return nameW, destW
}

func MoveLine(w io.Writer, name, dest, rule, age, size string, nameW, destW int) {
	fmt.Fprintf(w, "  %-*s %s→%s %-*s  %s%s · %s · %s%s\n",
		nameW, Trunc(name, nameW),
		Green, Reset,
		destW, Trunc(dest, destW),
		Dim, rule, age, size, Reset,
	)
}

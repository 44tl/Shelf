package shelf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/44tl/Shelf/internal/ui"
)

func StateDir() string {
	if s := os.Getenv("XDG_STATE_HOME"); strings.TrimSpace(s) != "" {
		return filepath.Join(s, "shelf")
	}
	if runtime.GOOS == "windows" {
		if base, err := os.UserConfigDir(); err == nil {
			return filepath.Join(base, "shelf")
		}
		return ".shelf-state"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".shelf-state"
	}
	return filepath.Join(home, ".local", "state", "shelf")
}

func TrashDir() string {
	return filepath.Join(StateDir(), "trash")
}

func PurgeExpiredTrash(out io.Writer, retention time.Duration) int {
	if retention <= 0 {
		return 0
	}
	dir := TrashDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	now := time.Now()
	purged := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") || !e.Type().IsRegular() {
			continue
		}
		info, err := e.Info()
		if err != nil || now.Sub(info.ModTime()) < retention {
			continue
		}
		if os.Remove(filepath.Join(dir, e.Name())) == nil {
			purged++
		}
	}
	if purged > 0 {
		fmt.Fprintf(out, "%s  [shelf] permanently purged %d expired file%s from trash%s\n",
			ui.Dim, purged, plural(purged), ui.Reset)
	}
	return purged
}

func DestDisplay(m Move) string {
	if m.Delete {
		return "shelf-trash · auto-purges"
	}
	return ui.Shorten(filepath.Dir(m.To))
}

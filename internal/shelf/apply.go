package shelf

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/44tl/Shelf/internal/ui"
)

func Apply(out io.Writer, moves []Move, jnl *Journal, dry bool, keepDeleted time.Duration) (moved int, bytes int64) {
	nameW, destW := layoutWidths(moves)
	taken := map[string]bool{}
	journalWarned := false

	if !dry {
		PurgeExpiredTrash(out, keepDeleted)
	}

	for _, m := range moves {
		if !isRegularFile(m.From) {
			continue
		}

		if dry {
			ui.MoveLine(out,
				filepath.Base(m.From),
				DestDisplay(m),
				m.Rule,
				ui.Age(m.Age),
				ui.HumanBytes(m.Size),
				nameW, destW,
			)
			moved++
			bytes += m.Size
			continue
		}

		if err := os.MkdirAll(filepath.Dir(m.To), 0o755); err != nil {
			failLine(out, filepath.Base(m.From), err)
			continue
		}

		m.To = reserveTarget(m.To, taken)

		if err := moveTo(m.From, m.To); err != nil {
			if !errors.Is(err, os.ErrExist) {
				failLine(out, filepath.Base(m.From), err)
				continue
			}
			m.To = uniqueDest(filepath.Dir(m.To), filepath.Base(m.To), taken)
			if err := moveTo(m.From, m.To); err != nil {
				failLine(out, filepath.Base(m.From), err)
				continue
			}
		}

		if err := jnl.Append(m); err != nil && !journalWarned {
			journalWarned = true
			fmt.Fprintf(out, "  %s! undo journal unavailable (%v) — moves continue without an undo trail%s\n",
				ui.Yellow, err, ui.Reset)
		}
		ui.MoveLine(out,
			filepath.Base(m.From),
			DestDisplay(m),
			m.Rule,
			ui.Age(m.Age),
			ui.HumanBytes(m.Size),
			nameW, destW,
		)
		moved++
		bytes += m.Size
	}
	return moved, bytes
}

func failLine(out io.Writer, name string, err error) {
	fmt.Fprintf(out, "  %s✗ %s (%v)%s\n", ui.Red, name, err, ui.Reset)
}

func isRegularFile(p string) bool {
	fi, err := os.Lstat(p)
	return err == nil && fi.Mode().IsRegular()
}

func reserveTarget(target string, taken map[string]bool) string {
	for i := 0; i < 8; i++ {
		if _, err := os.Lstat(target); err != nil {
			return target
		}
		target = uniqueDest(filepath.Dir(target), filepath.Base(target), taken)
	}
	return target
}

func moveTo(from, to string) error {
	linkErr := os.Link(from, to)
	switch {
	case linkErr == nil:
		if err := os.Remove(from); err != nil {
			os.Remove(to)
			return err
		}
		return nil
	case errors.Is(linkErr, os.ErrExist):
		return linkErr
	default:
		return copyDelete(from, to)
	}
}

func copyDelete(from, to string) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}

	fi, err := src.Stat()
	if err != nil {
		src.Close()
		return err
	}
	if !fi.Mode().IsRegular() {
		src.Close()
		return errors.New("shelf: source changed mid-move")
	}

	dst, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fi.Mode().Perm())
	if err != nil {
		src.Close()
		return err
	}

	if _, err = io.Copy(dst, src); err != nil {
		src.Close()
		dst.Close()
		os.Remove(to)
		return err
	}
	if err = dst.Sync(); err != nil {
		src.Close()
		dst.Close()
		os.Remove(to)
		return err
	}
	if err = dst.Close(); err != nil {
		src.Close()
		os.Remove(to)
		return err
	}

	src.Close()

	os.Chmod(to, fi.Mode().Perm())
	os.Chtimes(to, fi.ModTime(), fi.ModTime())

	if err := os.Remove(from); err != nil {
		os.Remove(to)
		return err
	}
	return nil
}

func layoutWidths(moves []Move) (nameW, destW int) {
	names := make([]string, len(moves))
	dests := make([]string, len(moves))
	for i, m := range moves {
		names[i] = filepath.Base(m.From)
		dests[i] = DestDisplay(m)
	}
	return ui.Widths(names, dests)
}

package shelf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/44tl/Shelf/internal/ui"
)

func Apply(out io.Writer, moves []Move, jnl *Journal, dry bool) (moved int, bytes int64) {
	nameW, destW := layoutWidths(moves)
	for _, m := range moves {
		if !dry {
			if err := os.MkdirAll(filepath.Dir(m.To), 0o755); err != nil {
				fmt.Fprintf(out, "  %s✗ %s (%v)%s\n", ui.Red, filepath.Base(m.From), err, ui.Reset)
				continue
			}
			if err := os.Rename(m.From, m.To); err != nil {
				fmt.Fprintf(out, "  %s✗ %s (%v)%s\n", ui.Red, filepath.Base(m.From), err, ui.Reset)
				continue
			}
			jnl.Append(m)
		} else if _, err := os.Lstat(m.From); err != nil {
			continue
		}
		ui.MoveLine(out,
			filepath.Base(m.From),
			ui.Shorten(filepath.Dir(m.To)),
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

func layoutWidths(moves []Move) (nameW, destW int) {
	names := make([]string, len(moves))
	dests := make([]string, len(moves))
	for i, m := range moves {
		names[i] = filepath.Base(m.From)
		dests[i] = ui.Shorten(filepath.Dir(m.To))
	}
	return ui.Widths(names, dests)
}

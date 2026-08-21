package shelf

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	fsnotify "github.com/fsnotify/fsnotify"

	"github.com/44tl/Shelf/internal/config"
	"github.com/44tl/Shelf/internal/ui"
)

const (
	debounceDelay = 800 * time.Millisecond
	settleWindow  = 20 * time.Second
)

func Watch(out io.Writer, root string, cfg *config.Config, jnl *Journal) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	if err := watcher.Add(root); err != nil {
		return fmt.Errorf("shelf: watching %s: %w", root, err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	fire := make(chan struct{}, 1)
	var timer *time.Timer
	kick := func() {
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(debounceDelay, func() { fire <- struct{}{} })
	}

	fmt.Fprintf(out, "%sshelf%s is watching %s — %sctrl-c to stop%s\n",
		ui.Bold, ui.Reset, ui.Shorten(root), ui.Dim, ui.Reset)
	runOnce(out, root, cfg, jnl)

	for {
		select {
		case e, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			base := filepath.Base(e.Name)
			if strings.HasPrefix(base, ".") || excluded(base) {
				continue
			}
			if e.Has(fsnotify.Create) || e.Has(fsnotify.Write) ||
				e.Has(fsnotify.Remove) || e.Has(fsnotify.Rename) {
				kick()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(out, "%sshelf: %v%s\n", ui.Yellow, err, ui.Reset)
		case <-fire:
			runOnce(out, root, cfg, jnl)
		case <-sigs:
			fmt.Fprintf(out, "\n%sshelf signed off. Downloads stays tidy.%s\n", ui.Dim, ui.Reset)
			return nil
		}
	}
}

func runOnce(out io.Writer, root string, cfg *config.Config, jnl *Journal) {
	moves, err := Plan(root, cfg, settleWindow, time.Now())
	if err != nil {
		fmt.Fprintf(out, "%sshelf: %v%s\n", ui.Yellow, err, ui.Reset)
		return
	}
	if len(moves) == 0 {
		return
	}
	jnl.StartRun()
	moved, bytes := Apply(out, moves, jnl, false)
	if moved > 0 {
		ts := time.Now().Format("15:04:05")
		fmt.Fprintf(out, "%s[%s] filed %d file%s (%s)%s\n",
			ui.Dim, ts, moved, plural(moved), ui.HumanBytes(bytes), ui.Reset)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

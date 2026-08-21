package shelf

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/44tl/Shelf/internal/ui"
)

type Journal struct {
	path string
	run  string
}

type entry struct {
	Run  string    `json:"run"`
	Time time.Time `json:"time"`
	From string    `json:"from"`
	To   string    `json:"to"`
}

func DefaultJournalPath() string {
	return filepath.Join(StateDir(), "journal.jsonl")
}

func OpenJournal(path string) (*Journal, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("shelf: journal: %w", err)
	}
	return &Journal{path: path}, nil
}

func (j *Journal) StartRun() {
	j.run = fmt.Sprintf("%d", time.Now().UnixNano())
}

func (j *Journal) Append(m Move) error {
	if j.run == "" {
		j.StartRun()
	}
	f, err := os.OpenFile(j.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	e := entry{Run: j.run, Time: time.Now(), From: m.From, To: m.To}
	b, err := json.Marshal(&e)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

func (j *Journal) Undo(out io.Writer) (int, error) {
	group, rest, err := j.lastRun()
	if err != nil {
		return 0, err
	}
	if len(group) == 0 {
		fmt.Fprintf(out, "%snothing to undo — the journal is empty.%s\n", ui.Dim, ui.Reset)
		return 0, nil
	}

	restored := 0
	var kept []entry
	for i := len(group) - 1; i >= 0; i-- {
		e := group[i]
		if _, err := os.Lstat(e.To); err != nil {
			fmt.Fprintf(out, "%s  ? %s — destination vanished, giving up on it%s\n", ui.Yellow, ui.Trunc(filepath.Base(e.To), 60), ui.Reset)
			continue
		}
		if _, err := os.Lstat(e.From); err == nil {
			fmt.Fprintf(out, "%s  ! %s — source already occupied, kept for a later undo%s\n", ui.Yellow, ui.Trunc(filepath.Base(e.From), 60), ui.Reset)
			kept = append(kept, e)
			continue
		}
		os.MkdirAll(filepath.Dir(e.From), 0o755)
		if err := os.Rename(e.To, e.From); err != nil {
			if cerr := copyDelete(e.To, e.From); cerr != nil {
				fmt.Fprintf(out, "%s  ✗ %s (%v) — kept for a later undo%s\n", ui.Red, ui.Trunc(filepath.Base(e.To), 60), cerr, ui.Reset)
				kept = append(kept, e)
				continue
			}
		}
		restored++
		fmt.Fprintf(out, "  %s←%s %s\n", ui.Cyan, ui.Reset, ui.Shorten(e.From))
	}

	if len(kept) > 0 {
		rest = append(rest, kept...)
	}
	if err := rewrite(j.path, rest); err != nil {
		return restored, err
	}
	return restored, nil
}

func (j *Journal) lastRun() ([]entry, []entry, error) {
	f, err := os.Open(j.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	defer f.Close()

	var all []entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e entry
		if json.Unmarshal([]byte(line), &e) == nil && e.From != "" {
			all = append(all, e)
		}
	}
	if len(all) == 0 {
		return nil, nil, nil
	}
	lastID := all[len(all)-1].Run
	var group, rest []entry
	for _, e := range all {
		if e.Run == lastID {
			group = append(group, e)
		} else {
			rest = append(rest, e)
		}
	}
	return group, rest, nil
}

func rewrite(path string, entries []entry) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, e := range entries {
		b, err := json.Marshal(&e)
		if err == nil {
			w.Write(b)
			w.WriteByte('\n')
		}
	}
	w.Flush()
	f.Close()
	return os.Rename(tmp, path)
}

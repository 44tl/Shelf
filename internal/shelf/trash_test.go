package shelf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/44tl/Shelf/internal/config"
)

func junkCfg(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Parse([]byte(`
keep_deleted: 30d
rules:
  - name: Junk
    match: ["*.junk"]
    older_than: 1d
    delete: true
`))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestDeleteRulePlansIntoPrivateTrash(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	root := t.TempDir()

	src := touch(t, root, "garbage.junk", 5*24*time.Hour, 42)
	os.WriteFile(filepath.Join(root, "fresh.junk"), []byte("x"), 0o644)

	moves, err := Plan(root, junkCfg(t), 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(moves) != 1 {
		t.Fatalf("planned %d moves, want only the aged one: %+v", len(moves), moves)
	}
	m := moves[0]
	if !m.Delete {
		t.Error("move is not flagged as a delete")
	}
	if want := filepath.Join(state, "shelf", "trash"); filepath.Dir(m.To) != want {
		t.Errorf("dest %q is not inside trash dir %q", m.To, want)
	}
	if filepath.Base(m.To) == "garbage.junk" {
		t.Error("trash name should be timestamp-prefixed to avoid collisions")
	}
	_ = src
}

func TestDeleteApplyThenUndoBringsFileBack(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()

	content := []byte("precious junk")
	src := filepath.Join(root, "old.junk")
	os.WriteFile(src, content, 0o644)
	old := time.Now().Add(-5 * 24 * time.Hour)
	os.Chtimes(src, old, old)

	jnl, err := OpenJournal(filepath.Join(t.TempDir(), "journal.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	moves, err := Plan(root, junkCfg(t), 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	jnl.StartRun()
	moved, _ := Apply(&out, moves, jnl, false, 30*24*time.Hour)
	if moved != 1 {
		t.Fatalf("moved %d, want 1 (out: %s)", moved, out.String())
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Fatal("source still present after delete-apply")
	}

	n, err := jnl.Undo(&out)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("undo restored %d, want 1", n)
	}
	got, err := os.ReadFile(src)
	if err != nil || string(got) != string(content) {
		t.Fatalf("restored content = %q (%v), want %q", got, err, content)
	}
}

func TestPurgeExpiredTrashOnlyRemovesStaleFiles(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	trash := TrashDir()
	os.MkdirAll(trash, 0o700)

	stale := filepath.Join(trash, "old-thing.bin")
	fresh := filepath.Join(trash, "new-thing.bin")
	os.WriteFile(stale, []byte("a"), 0o644)
	os.WriteFile(fresh, []byte("b"), 0o644)
	old := time.Now().Add(-31 * 24 * time.Hour)
	os.Chtimes(stale, old, old)

	var out bytes.Buffer
	if n := PurgeExpiredTrash(&out, 30*24*time.Hour); n != 1 {
		t.Fatalf("purged %d, want 1", n)
	}
	if _, err := os.Lstat(stale); !os.IsNotExist(err) {
		t.Error("stale file survived the purge")
	}
	if _, err := os.Lstat(fresh); err != nil {
		t.Error("fresh file was purged early")
	}

	PurgeExpiredTrash(&out, 0)
	if _, err := os.Lstat(fresh); err != nil {
		t.Error("retention=0 disabled purge but deleted anyway")
	}
}

func TestPreviewNeverPurgesTrash(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	trash := TrashDir()
	os.MkdirAll(trash, 0o700)
	stale := filepath.Join(trash, "waiting.bin")
	os.WriteFile(stale, []byte("a"), 0o644)
	old := time.Now().Add(-31 * 24 * time.Hour)
	os.Chtimes(stale, old, old)

	var out bytes.Buffer
	jnl, _ := OpenJournal(filepath.Join(t.TempDir(), "journal.jsonl"))
	Apply(&out, nil, jnl, true, 30*24*time.Hour)

	if _, err := os.Lstat(stale); err != nil {
		t.Fatal("preview purged the trash; previews must touch nothing")
	}
}

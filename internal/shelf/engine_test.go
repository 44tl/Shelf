package shelf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/44tl/Shelf/internal/config"
)

func smallCfg(dir string) *config.Config {
	yaml := `
rules:
  - name: Pics
    match: ["*.png"]
    to: ` + filepath.Join(dir, "pics") + `
  - name: Docs
    match: ["*.pdf"]
    older_than: 10d
    to: ` + filepath.Join(dir, "docs") + `
`
	cfg, err := config.Parse([]byte(yaml))
	if err != nil {
		panic(err)
	}
	return cfg
}

func touch(t *testing.T, dir, name string, age time.Duration, size int) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-age)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPlanMatchesAndFilters(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()
	cfg := smallCfg(dest)

	touch(t, root, "a.png", 5*24*time.Hour, 100)
	touch(t, root, "b.png", 1*time.Hour, 100)
	touch(t, root, "old.pdf", 40*24*time.Hour, 50)
	touch(t, root, "new.pdf", 1*time.Hour, 50)
	touch(t, root, ".hidden.png", 40*24*time.Hour, 10)
	touch(t, root, "setup.crdownload", 40*24*time.Hour, 999999)

	moves, err := Plan(root, cfg, 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, m := range moves {
		got[filepath.Base(m.From)] = m.Rule
	}
	want := map[string]string{
		"a.png":   "Pics",
		"b.png":   "Pics",
		"old.pdf": "Docs",
	}
	for f, r := range want {
		if got[f] != r {
			t.Errorf("%s: expected rule %q, got %q", f, r, got[f])
		}
	}
	for _, f := range []string{"new.pdf", ".hidden.png", "setup.crdownload"} {
		if _, ok := got[f]; ok {
			t.Errorf("%s should not be planned", f)
		}
	}
}

func TestPlanResolvesNameConflicts(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()

	os.MkdirAll(filepath.Join(dest, "pics"), 0o755)
	os.WriteFile(filepath.Join(dest, "pics", "a.png"), []byte("x"), 0o644)

	cfg := smallCfg(dest)
	touch(t, root, "a.png", 24*time.Hour, 1)
	touch(t, root, "a.png.bak", 24*time.Hour, 1)

	moves, err := Plan(root, cfg, 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(moves) != 1 {
		t.Fatalf("expected 1 move, got %d: %+v", len(moves), moves)
	}
	if filepath.Base(moves[0].To) != "a (2).png" {
		t.Errorf("conflict not resolved: %s", moves[0].To)
	}
}

func TestApplyThenUndoRestoresEverything(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()
	cfg := smallCfg(dest)

	srcA := touch(t, root, "a.png", 24*time.Hour, 11)
	srcB := touch(t, root, "old.pdf", 40*24*time.Hour, 22)

	jnl, err := OpenJournal(filepath.Join(t.TempDir(), "state", "journal.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	moves, err := Plan(root, cfg, 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	jnl.StartRun()
	moved, bytes := Apply(&out, moves, jnl, false)

	if moved != 2 || bytes != 33 {
		t.Fatalf("apply reported %d files / %d bytes, want 2 / 33", moved, bytes)
	}
	for _, src := range []string{srcA, srcB} {
		if _, err := os.Lstat(src); !os.IsNotExist(err) {
			t.Errorf("%s still exists after apply", src)
		}
	}
	movedA := filepath.Join(dest, "pics", "a.png")
	movedB := filepath.Join(dest, "docs", "old.pdf")
	for _, dst := range []string{movedA, movedB} {
		if _, err := os.Lstat(dst); err != nil {
			t.Errorf("%s missing after apply: %v", dst, err)
		}
	}

	restored, err := jnl.Undo(&out)
	if err != nil {
		t.Fatal(err)
	}
	if restored != 2 {
		t.Fatalf("undo restored %d files, want 2", restored)
	}
	for _, src := range []string{srcA, srcB} {
		if _, err := os.Lstat(src); err != nil {
			t.Errorf("%s not restored: %v", src, err)
		}
	}
	for _, dst := range []string{movedA, movedB} {
		if _, err := os.Lstat(dst); !os.IsNotExist(err) {
			t.Errorf("%s still present after undo", dst)
		}
	}

	again, _ := jnl.Undo(&out)
	if again != 0 {
		t.Errorf("second undo restored %d files, want 0", again)
	}
}

func TestUniqueDestAlsoChecksPlannedSet(t *testing.T) {
	planned := map[string]bool{}
	d := t.TempDir()

	a := uniqueDest(d, "f.txt", planned)
	b := uniqueDest(d, "f.txt", planned)
	c := uniqueDest(d, "f.txt", planned)

	if a == b || b == c || a == c {
		t.Errorf("planned destinations collided: %s %s %s", a, b, c)
	}
	if filepath.Base(b) != "f (2).txt" || filepath.Base(c) != "f (3).txt" {
		t.Errorf("unexpected numbering: %s %s", b, c)
	}
}

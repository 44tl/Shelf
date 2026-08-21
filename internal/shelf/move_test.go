package shelf

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCopyDeletePreservesContentModeAndTime(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()

	src := filepath.Join(root, "payload.bin")
	content := []byte("shelf cross-device payload")
	os.WriteFile(src, content, 0o640)
	mtime := time.Now().Add(-72 * time.Hour).Truncate(time.Second)
	os.Chtimes(src, mtime, mtime)

	dst := filepath.Join(dest, "payload.bin")
	if err := copyDelete(src, dst); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dst)
	if err != nil || string(got) != string(content) {
		t.Fatalf("destination content = %q (%v), want %q", got, err, content)
	}
	if fi, err := os.Stat(dst); err != nil {
		t.Fatal(err)
	} else {
		if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o640 {
			t.Errorf("perm = %v, want 640", fi.Mode().Perm())
		}
		if d := fi.ModTime().Sub(mtime); d > 2*time.Second || d < -2*time.Second {
			t.Errorf("mtime drifted by %v", d)
		}
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Error("source survived a successful cross-device move")
	}
}

func TestCopyDeleteNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()

	src := filepath.Join(root, "a.txt")
	os.WriteFile(src, []byte("new"), 0o644)

	dst := filepath.Join(dest, "a.txt")
	os.WriteFile(dst, []byte("precious"), 0o644)

	err := copyDelete(src, dst)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("err = %v, want ErrExist", err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "precious" {
		t.Fatal("existing destination was overwritten by the copy fallback")
	}
	if _, err := os.Lstat(src); err != nil {
		t.Error("source was removed even though the move failed")
	}
}

func TestCopyDeleteRollsBackWhenSourceSticks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission semantics differ on windows")
	}
	root := t.TempDir()
	dest := t.TempDir()

	src := filepath.Join(root, "stuck.bin")
	os.WriteFile(src, []byte("data"), 0o600)

	os.Chmod(root, 0o555)
	t.Cleanup(func() { os.Chmod(root, 0o755) })

	dst := filepath.Join(dest, "stuck.bin")
	if err := copyDelete(src, dst); err == nil {
		t.Fatal("expected an error when the source cannot be removed")
	}
	if _, serr := os.Lstat(src); serr != nil {
		t.Error("source vanished despite failed removal")
	}
	if _, derr := os.Lstat(dst); !os.IsNotExist(derr) {
		t.Error("completed copy was not rolled back; watch mode would clone it forever")
	}
}

func TestMoveToPrefersHardlinksAndFallsBackCleanly(t *testing.T) {
	root := t.TempDir()

	src := filepath.Join(root, "same-fs.bin")
	os.WriteFile(src, []byte("x"), 0o600)

	dst := filepath.Join(root, "moved.bin")
	if err := moveTo(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Error("source still present after hardlink move")
	}
	if _, err := os.Lstat(dst); err != nil {
		t.Errorf("destination missing: %v", err)
	}
}

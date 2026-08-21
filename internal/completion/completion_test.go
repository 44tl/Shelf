package completion

import (
	"strings"
	"testing"
)

func TestAllShellsResolve(t *testing.T) {
	for _, shell := range Shells() {
		script, err := Script(shell)
		if err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		if len(script) < 50 {
			t.Errorf("%s script suspiciously short (%d bytes)", shell, len(script))
		}
		if !strings.Contains(script, "shelf") {
			t.Errorf("%s script never mentions the command", shell)
		}
	}
}

func TestPwshAlias(t *testing.T) {
	a, err := Script("pwsh")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Script("powershell")
	if a != b {
		t.Error("pwsh and powershell should yield the same script")
	}
}

func TestUnknownShellFails(t *testing.T) {
	if _, err := Script("cmd.exe"); err == nil {
		t.Fatal("expected an error for an unknown shell")
	}
}

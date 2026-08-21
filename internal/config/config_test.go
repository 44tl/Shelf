package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultRulesAreValid(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatalf("Default() error: %v", err)
	}
	if len(cfg.Rules) == 0 {
		t.Fatal("built-in ruleset is empty")
	}
	for _, r := range cfg.Rules {
		if r.Dest == "" || len(r.Match) == 0 {
			t.Errorf("rule %q did not resolve (dest=%q match=%v)", r.Name, r.Dest, r.Match)
		}
		if !filepath.IsAbs(r.Dest) {
			t.Errorf("rule %q dest is not absolute: %s", r.Name, r.Dest)
		}
	}
}

func TestMatchFirstRuleWins(t *testing.T) {
	cfg, _ := Default()
	rule, err := Match(cfg, "Screenshot from 2026-01-01.png")
	if err != nil {
		t.Fatal(err)
	}
	if rule == nil || rule.Name != "Screenshots" {
		t.Fatalf("expected Screenshots rule, got %+v", rule)
	}

	rule, _ = Match(cfg, "SETUP.EXE")
	if rule == nil || rule.Name != "Installers" {
		t.Fatalf("expected Installers rule for SETUP.EXE, got %+v", rule)
	}

	rule, _ = Match(cfg, "mysteryfile.xyz")
	if rule != nil {
		t.Fatalf("unmatched file should return nil, got %+v", rule)
	}
}

func TestParseRejectsBrokenRules(t *testing.T) {
	cases := map[string]string{
		"missing to": `
rules:
  - name: X
    match: ["*.png"]
`,
		"missing match": `
rules:
  - name: X
    to: ~/somewhere
`,
		"bad pattern": `
rules:
  - name: X
    match: ["[unclosed"]
    to: ~/somewhere
`,
		"bad duration": `
rules:
  - name: X
    match: ["*.png"]
    older_than: "soon"
    to: ~/somewhere
`,
		"delete with to": `
rules:
  - name: X
    match: ["*.png"]
    older_than: 7d
    delete: true
    to: ~/somewhere
`,
		"delete without older_than": `
rules:
  - name: X
    match: ["*.png"]
    delete: true
`,
		"keep_deleted below floor": `
keep_deleted: 20m
rules:
  - name: X
    match: ["*.png"]
    to: ~/somewhere
`,
		"no rules": `
rules: []
`,
	}
	for name, text := range cases {
		if _, err := Parse([]byte(text)); err == nil {
			t.Errorf("%s: expected error, got none", name)
		}
	}
}

func TestExpandHome(t *testing.T) {
	got, err := ExpandHome("~/notes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "notes") || strings.HasPrefix(got, "~") {
		t.Errorf("ExpandHome did not resolve ~: %s", got)
	}
	plain, _ := ExpandHome("/tmp/x")
	if plain != "/tmp/x" {
		t.Errorf("plain path changed: %s", plain)
	}
}

func TestParseInResolvesRelativeDestAgainstBaseDir(t *testing.T) {
	base := t.TempDir()
	text := `
rules:
  - name: Rel
    match: ["*.png"]
    to: sorted-files
`
	cfg, err := ParseIn(base, []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, "sorted-files")
	if cfg.Rules[0].Dest != want {
		t.Errorf("relative dest = %q, want %q", cfg.Rules[0].Dest, want)
	}

	cwdCfg, err := Parse([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(cwdCfg.Rules[0].Dest) {
		t.Errorf("Parse should still produce an absolute dest, got %q", cwdCfg.Rules[0].Dest)
	}
}

func TestDeleteRuleValidation(t *testing.T) {
	cfg, err := Parse([]byte(`
rules:
  - name: Junk
    match: ["*.tmp"]
    older_than: 90d
    delete: true
`))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Rules[0].Delete || cfg.Rules[0].Dest != "" {
		t.Errorf("delete rule parsed wrong: %+v", cfg.Rules[0])
	}
	if cfg.KeepDel != 30*24*time.Hour {
		t.Errorf("default retention = %v, want 30d", cfg.KeepDel)
	}

	cfg2, err := Parse([]byte(`
keep_deleted: 45d
rules:
  - name: X
    match: ["*.png"]
    to: ~/somewhere
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.KeepDel != 45*24*time.Hour {
		t.Errorf("keep_deleted = %v, want 45d", cfg2.KeepDel)
	}
}

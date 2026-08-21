package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Name      string   `yaml:"name"`
	Match     []string `yaml:"match"`
	OlderThan string   `yaml:"older_than,omitempty"`
	To        string   `yaml:"to"`
	Delete    bool     `yaml:"delete,omitempty"`

	Older time.Duration
	Dest  string
	Pats  []string
}

type Config struct {
	Rules       []Rule `yaml:"rules"`
	KeepDeleted string `yaml:"keep_deleted,omitempty"`

	KeepDel time.Duration
}

const DefaultText = `# shelf — rules for giving every download a home.
#
# Rules are checked top to bottom; the first match wins.
# "match" globs are case-insensitive and match the file name.
# "older_than" is optional; young files are left alone.
# "to" may start with ~ and is created on demand.
#
# Durations accept: 30m, 12h, 7d, 2w, or combos like 1w2d.

rules:
  - name: Screenshots
    match: ["screenshot*", "*.png", "*.jpg", "*.jpeg", "*.webp", "*.gif", "*.heic"]
    older_than: 7d
    to: ~/Pictures/Shelf

  - name: Documents
    match: ["*.pdf", "*.doc*", "*.xls*", "*.ppt*", "*.odt", "*.txt", "*.md", "*.csv", "*.epub"]
    older_than: 14d
    to: ~/Documents/Shelf

  - name: Archives
    match: ["*.zip", "*.tar", "*.tar.*", "*.tgz", "*.rar", "*.7z", "*.gz"]
    older_than: 30d
    to: ~/Archives/Shelf

  - name: Video
    match: ["*.mp4", "*.mkv", "*.mov", "*.avi", "*.webm"]
    older_than: 30d
    to: ~/Videos/Shelf

  - name: Music
    match: ["*.mp3", "*.flac", "*.wav", "*.m4a", "*.ogg"]
    older_than: 30d
    to: ~/Music/Shelf

  - name: Installers
    match: ["*.exe", "*.msi", "*.dmg", "*.pkg", "*.deb", "*.rpm", "*.appimage", "*.iso"]
    older_than: 30d
    to: ~/Downloads/Installers

# True junk can be deleted instead of moved — it goes to shelf's private
# trash first, comes back with "shelf --undo", and is purged for real
# after keep_expired (30 days by default). "older_than" is required for
# delete rules, so fresh files are never touched.

#  - name: True junk
#    match: ["*.tmp", "*.log"]
#    older_than: 90d
#    delete: true

# keep_deleted: 30d
`

func Default() (*Config, error) { return Parse([]byte(DefaultText)) }

func StarterText() string { return DefaultText }

func Path() string {
	if x := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); x != "" {
		return filepath.Join(x, "shelf", "shelf.yaml")
	}
	if runtime.GOOS == "windows" {
		if base, err := os.UserConfigDir(); err == nil {
			return filepath.Join(base, "shelf", "shelf.yaml")
		}
		return "shelf.yaml"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "shelf.yaml"
	}
	return filepath.Join(home, ".config", "shelf", "shelf.yaml")
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("shelf: reading %s: %w", path, err)
	}
	base := ""
	if abs, aerr := filepath.Abs(path); aerr == nil {
		base = filepath.Dir(abs)
	}
	cfg, err := ParseIn(base, b)
	if err != nil {
		return nil, fmt.Errorf("shelf: in %s: %w", path, err)
	}
	return cfg, nil
}

func Parse(b []byte) (*Config, error) { return ParseIn("", b) }

func ParseIn(baseDir string, b []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	for i := range c.Rules {
		r := &c.Rules[i]
		if r.Name == "" {
			r.Name = fmt.Sprintf("rule %d", i+1)
		}
		if len(r.Match) == 0 {
			return nil, fmt.Errorf("rule %q has no \"match\" patterns", r.Name)
		}
		r.Pats = make([]string, len(r.Match))
		for j, pat := range r.Match {
			lowered := strings.ToLower(pat)
			if _, err := filepath.Match(lowered, ""); err != nil {
				return nil, fmt.Errorf("rule %q: bad pattern %q: %w", r.Name, pat, err)
			}
			r.Pats[j] = lowered
		}
		d, err := ParseDuration(r.OlderThan)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", r.Name, err)
		}
		r.Older = d

		if r.Delete {
			if strings.TrimSpace(r.To) != "" {
				return nil, fmt.Errorf("rule %q sets both \"to\" and \"delete\" — pick one", r.Name)
			}
			if r.Older <= 0 {
				return nil, fmt.Errorf("rule %q uses \"delete\" and must set \"older_than\" — fresh files are never deleted", r.Name)
			}
		} else {
			if strings.TrimSpace(r.To) == "" {
				return nil, fmt.Errorf("rule %q is missing \"to\"", r.Name)
			}
			dest, err := ExpandHome(r.To)
			if err != nil {
				return nil, fmt.Errorf("rule %q: %w", r.Name, err)
			}
			if !filepath.IsAbs(dest) && baseDir != "" {
				dest = filepath.Join(baseDir, dest)
			}
			abs, err := filepath.Abs(dest)
			if err != nil {
				return nil, fmt.Errorf("rule %q: %w", r.Name, err)
			}
			r.Dest = filepath.Clean(abs)
		}
	}
	if len(c.Rules) == 0 {
		return nil, errors.New(`no rules defined`)
	}
	c.KeepDel = 30 * 24 * time.Hour
	if strings.TrimSpace(c.KeepDeleted) != "" {
		kd, err := ParseDuration(c.KeepDeleted)
		if err != nil {
			return nil, fmt.Errorf("keep_deleted: %w", err)
		}
		if kd != 0 && kd < time.Hour {
			return nil, fmt.Errorf("keep_deleted %q is below the one-hour safety floor", c.KeepDeleted)
		}
		if kd > 0 {
			c.KeepDel = kd
		}
	}
	return &c, nil
}

func Match(c *Config, name string) (*Rule, error) {
	lower := strings.ToLower(name)
	for i := range c.Rules {
		r := &c.Rules[i]
		for _, pat := range r.Pats {
			ok, err := filepath.Match(pat, lower)
			if err != nil {
				return nil, fmt.Errorf("rule %q: bad pattern %q: %w", r.Name, pat, err)
			}
			if ok {
				return r, nil
			}
		}
	}
	return nil, nil
}

func ExpandHome(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.New("~ used but home directory is unknown")
		}
		if p == "~" {
			return home, nil
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}

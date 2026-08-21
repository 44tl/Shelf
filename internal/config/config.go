package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Name      string   `yaml:"name"`
	Match     []string `yaml:"match"`
	OlderThan string   `yaml:"older_than,omitempty"`
	To        string   `yaml:"to"`

	Older time.Duration
	Dest  string
}

type Config struct {
	Rules []Rule `yaml:"rules"`
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
`

func Default() (*Config, error) { return Parse([]byte(DefaultText)) }

func StarterText() string { return DefaultText }

func Path() string {
	base, err := os.UserConfigDir()
	if err != nil {
		if home, herr := os.UserHomeDir(); herr == nil {
			return filepath.Join(home, ".config", "shelf", "shelf.yaml")
		}
		return "shelf.yaml"
	}
	return filepath.Join(base, "shelf", "shelf.yaml")
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("shelf: reading %s: %w", path, err)
	}
	cfg, err := Parse(b)
	if err != nil {
		return nil, fmt.Errorf("shelf: in %s: %w", path, err)
	}
	return cfg, nil
}

func Parse(b []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	for i := range c.Rules {
		r := &c.Rules[i]
		if r.Name == "" {
			r.Name = fmt.Sprintf("rule %d", i+1)
		}
		if len(r.Match) == 0 {
			return nil, fmt.Errorf("rule %q has no \"match\" patterns", r.Name)
		}
		d, err := ParseDuration(r.OlderThan)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", r.Name, err)
		}
		r.Older = d
		if strings.TrimSpace(r.To) == "" {
			return nil, fmt.Errorf("rule %q is missing \"to\"", r.Name)
		}
		dest, err := ExpandHome(r.To)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", r.Name, err)
		}
		abs, err := filepath.Abs(dest)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", r.Name, err)
		}
		r.Dest = filepath.Clean(abs)
	}
	if len(c.Rules) == 0 {
		return nil, errors.New(`no rules defined`)
	}
	return &c, nil
}

func Match(c *Config, name string) (*Rule, error) {
	lower := strings.ToLower(name)
	for i := range c.Rules {
		r := &c.Rules[i]
		for _, pat := range r.Match {
			ok, err := filepath.Match(strings.ToLower(pat), lower)
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

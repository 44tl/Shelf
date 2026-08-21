package shelf

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/44tl/Shelf/internal/config"
)

type Move struct {
	From string
	To   string
	Rule string
	Size int64
	Age  time.Duration
}

var transientExts = []string{
	".crdownload", ".part", ".download", ".tmp", ".swp", ".partial",
}

func excluded(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range transientExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	for _, pat := range []string{"~", ".bak"} {
		if strings.HasSuffix(lower, pat) {
			return true
		}
	}
	return false
}

func Plan(root string, cfg *config.Config, settle time.Duration, now time.Time) ([]Move, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("shelf: reading %s: %w", root, err)
	}

	rootClean := filepath.Clean(root)
	planned := map[string]bool{}
	var moves []Move

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !e.Type().IsRegular() {
			continue
		}
		if excluded(name) {
			continue
		}
		rule, err := config.Match(cfg, name)
		if err != nil {
			return nil, err
		}
		if rule == nil {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		age := now.Sub(info.ModTime())
		if rule.Older > 0 && age < rule.Older {
			continue
		}
		if settle > 0 && age < settle {
			continue
		}
		if rule.Dest == rootClean {
			continue
		}
		dest := uniqueDest(rule.Dest, name, planned)
		moves = append(moves, Move{
			From: filepath.Join(rootClean, name),
			To:   dest,
			Rule: rule.Name,
			Size: info.Size(),
			Age:  age,
		})
	}

	sort.Slice(moves, func(i, j int) bool { return moves[i].From < moves[j].From })
	return moves, nil
}

func uniqueDest(dir, base string, planned map[string]bool) string {
	ext := filepath.Ext(base)
	stem := base[:len(base)-len(ext)]
	for i := 1; i <= 999; i++ {
		cand := base
		if i > 1 {
			cand = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		full := filepath.Join(dir, cand)
		_, err := os.Lstat(full)
		if os.IsNotExist(err) && !planned[full] {
			planned[full] = true
			return full
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%s.%d%s", stem, time.Now().UnixNano(), ext))
}

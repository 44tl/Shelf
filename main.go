package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/44tl/Shelf/internal/config"
	"github.com/44tl/Shelf/internal/shelf"
	"github.com/44tl/Shelf/internal/ui"
)

var version = "dev"

const usageText = `shelf — give every download a home.

Usage:
  shelf [path]            preview what would move (default: ~/Downloads)
  shelf --apply [path]    actually move the files
  shelf --watch [path]    keep organizing as files arrive
  shelf --undo            revert the most recent applied run
  shelf --init            write a starter rules file to edit

Flags:
  --config path           use a specific rules file
  --version               print the version

Preview is the default: without --apply, nothing is ever touched.

Rules live in ~/.config/shelf/shelf.yaml — plain globs, plain words:

  - name: Invoices
    match: ["*invoice*.pdf"]
    older_than: 30d
    to: ~/Documents/Invoices
`

func main() {
	ui.Init()
	os.Exit(run())
}

func run() int {
	args := reorderArgs(os.Args[1:])

	var (
		apply   bool
		watch   bool
		undo    bool
		initCfg bool
		showVer bool
		cfgPath string
	)

	fsLike := map[string]*bool{
		"--apply":   &apply,
		"-a":        &apply,
		"--watch":   &watch,
		"-w":        &watch,
		"--undo":    &undo,
		"--init":    &initCfg,
		"--version": &showVer,
		"-v":        &showVer,
	}

	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			fmt.Print(usageText)
			return 0
		case a == "--config" || a == "-c":
			if i+1 >= len(args) {
				return fail("--config needs a path")
			}
			i++
			cfgPath = args[i]
		case strings.HasPrefix(a, "--config="):
			cfgPath = strings.TrimPrefix(a, "--config=")
		default:
			if p, ok := fsLike[a]; ok {
				*p = true
				continue
			}
			if strings.HasPrefix(a, "-") && a != "-" {
				return fail(fmt.Sprintf("unknown flag %q (try --help)", a))
			}
			positional = append(positional, a)
		}
	}

	if showVer {
		fmt.Printf("shelf %s\n", version)
		return 0
	}

	home, _ := os.UserHomeDir()
	ui.Home = home

	if initCfg {
		return writeStarterConfig()
	}
	if undo {
		return doUndo()
	}
	if apply && watch {
		return fail("--watch already applies moves; drop one of them")
	}

	root, err := resolveRoot(positional, home)
	if err != nil {
		return fail(err.Error())
	}

	cfg, source, err := loadConfig(cfgPath)
	if err != nil {
		return fail(err.Error())
	}

	jnl, err := shelf.OpenJournal(shelf.DefaultJournalPath())
	if err != nil {
		return fail(err.Error())
	}

	if watch {
		if err := shelf.Watch(os.Stdout, root, cfg, jnl); err != nil {
			return fail(err.Error())
		}
		return 0
	}

	return previewOrApply(root, cfg, source, jnl, apply)
}

func reorderArgs(args []string) []string {
	var flags, rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--config" || a == "-c" {
			flags = append(flags, a)
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
		} else {
			rest = append(rest, a)
		}
	}
	return append(flags, rest...)
}

func resolveRoot(positional []string, home string) (string, error) {
	if len(positional) > 1 {
		return "", fmt.Errorf("shelf: only one path at a time")
	}
	p := ""
	if len(positional) == 1 {
		p = positional[0]
	} else {
		p = filepath.Join(home, "Downloads")
	}
	expanded, err := config.ExpandHome(p)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("shelf: %v", err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("shelf: %s is not a directory", abs)
	}
	return abs, nil
}

func loadConfig(cfgPath string) (*config.Config, string, error) {
	if cfgPath != "" {
		cfg, err := config.Load(cfgPath)
		return cfg, cfgPath, err
	}
	p := config.Path()
	if _, err := os.Stat(p); err == nil {
		cfg, err := config.Load(p)
		return cfg, p, err
	}
	cfg, err := config.Default()
	return cfg, "(built-in defaults)", err
}

func writeStarterConfig() int {
	p := config.Path()
	if _, err := os.Stat(p); err == nil {
		fmt.Printf("%sshelf:%s rules already live at %s\n", ui.Yellow, ui.Reset, p)
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fail(err.Error())
	}
	if err := os.WriteFile(p, []byte(config.StarterText()), 0o644); err != nil {
		return fail(err.Error())
	}
	fmt.Printf("%swrote%s %s\nedit it to taste — shelf picks changes up on the next run.\n",
		ui.Green, ui.Reset, p)
	return 0
}

func doUndo() int {
	jnl, err := shelf.OpenJournal(shelf.DefaultJournalPath())
	if err != nil {
		return fail(err.Error())
	}
	n, err := jnl.Undo(os.Stdout)
	if err != nil {
		return fail(err.Error())
	}
	if n > 0 {
		fmt.Printf("\n%srestored %d file%s.%s\n", ui.Green, n, plural(n), ui.Reset)
	}
	return 0
}

func previewOrApply(root string, cfg *config.Config, source string, jnl *shelf.Journal, apply bool) int {
	moves, err := shelf.Plan(root, cfg, 0, time.Now())
	if err != nil {
		return fail(err.Error())
	}

	fmt.Printf("%s shelf %s·%s %s%s%s  %s%s%s\n\n",
		ui.Bold, ui.Reset, ui.Reset, ui.Dim, ui.Shorten(root), ui.Reset, ui.Dim, source, ui.Reset)

	if len(moves) == 0 {
		fmt.Printf("  %snothing to shelve — everything is already home.%s\n", ui.Green, ui.Reset)
		return 0
	}

	var bytes int64
	names := make([]string, 0, len(moves))
	dests := make([]string, 0, len(moves))
	for _, m := range moves {
		bytes += m.Size
		names = append(names, filepath.Base(m.From))
		dests = append(dests, ui.Shorten(filepath.Dir(m.To)))
	}

	shelf.Apply(os.Stdout, moves, jnl, !apply)

	fmt.Printf("\n  %d file%s · %s", len(moves), plural(len(moves)), ui.HumanBytes(bytes))
	if !apply {
		fmt.Printf(" — %spreview only, nothing moved%s\n", ui.Bold, ui.Reset)
		fmt.Printf("  %shappy? run again with --apply%s\n", ui.Dim, ui.Reset)
	} else {
		fmt.Printf(" — %sfiled.%s\n", ui.Green, ui.Reset)
		fmt.Printf("  %ssecond thoughts? shelf --undo%s\n", ui.Dim, ui.Reset)
	}
	return 0
}

func fail(msg string) int {
	fmt.Fprintf(os.Stderr, "%s%s%s\n", ui.Red, msg, ui.Reset)
	return 1
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

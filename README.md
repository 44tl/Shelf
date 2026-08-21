# shelf

![No AI](https://raw.githubusercontent.com/nuxy/no-ai-badge/master/badge.svg)

**Your Downloads folder, but livable.**

Every Downloads folder tells the same story. A screenshot from three weeks
ago you keep meaning to look at. An invoice you'll definitely file
"tomorrow". Four copies of the same PDF, a 6 GB iso from that one evening,
and something called `final_FINAL_v3(1).pdf`.

shelf is a small tool that quietly ends this. It looks at your folder,
tells you exactly which files it wants to move where — in plain words,
before touching anything — and then gives everything a home. Changed your
mind? `shelf --undo` puts it all back. Everything, always.

No AI, no cloud, no account, no telemetry. Just a little program that does
one chore so you don't have to.

```console
$ shelf
 shelf · ~/Downloads  (built-in defaults)

  Screenshot from 2026-07-01.png   → ~/Pictures/Shelf         Screenshots · 32d · 2.1 MB
  contract-signed.pdf              → ~/Documents/Shelf        Documents · 45d · 820 KB
  ubuntu-26.04.iso                 → ~/Downloads/Installers   Installers · 63d · 5.9 GB

  3 files · 9.0 GB — preview only, nothing moved
  happy? run again with --apply
```

## The ideas behind it

**Preview first, always.** Running `shelf` never moves a single byte. It
just shows its plan. You stay in charge; the `--apply` flag is the only
thing that touches your files.

**Undo is a feature, not an apology.** Every move is written to a small
journal on your disk. One command walks it back, newest file first, exactly
where things came from. Run it twice to go back further.

**Rules you could explain to a friend.** No magic categories, no guessing
what "smart sort" means this week. If you want invoices kept for a month
first, you write that:

```yaml
- name: Invoices
  match: ["*invoice*.pdf"]
  older_than: 30d
  to: ~/Documents/Invoices
```

That's the whole format. Globs, an age, a destination.

**It respects the living.** Dotfiles, directories, symlinks and half-downloaded
files (`.crdownload`, `.part`) are never touched. Files younger than a rule's
`older_than` are left alone to settle. Destinations are re-checked at the
moment of each move, so nothing that shows up in the meantime can ever be
overwritten — collisions become `file (2).ext` instead.

## Getting it

```sh
go install github.com/44tl/Shelf@latest
```

Living dangerously on the bleeding edge? `go install github.com/44tl/Shelf@main`
works even before the first release is tagged.

Prebuilt binaries for Linux, macOS and Windows are on the
[Releases](https://github.com/44tl/Shelf/releases) page.

## Living with it

```sh
shelf            # what would move in ~/Downloads?
shelf --apply    # go ahead
shelf --undo     # actually, no
shelf --init     # write ~/.config/shelf/shelf.yaml and make it yours
shelf --watch    # keep tidying as new files land (ctrl-c when done)
shelf ~/Desktop  # the other place clutter goes to die
```

Out of the box — with no config at all — screenshots drift to Pictures,
documents to Documents, archives to Archives, music and video to their
rooms, and old installers into `~/Downloads/Installers`. Everything else
stays put until you say otherwise.

Durations read the way you think: `45m`, `12h`, `7d`, `2w`, even `1w2d`.

## Tab completion

shelf can complete its own flags and folder paths:

```sh
# bash
mkdir -p ~/.local/share/bash-completion/completions
shelf --completion bash > ~/.local/share/bash-completion/completions/shelf

# zsh (before compinit in .zshrc)
mkdir -p ~/.zfunc && fpath=(~/.zfunc $fpath)
shelf --completion zsh > ~/.zfunc/_shelf

# fish
mkdir -p ~/.config/fish/completions
shelf --completion fish > ~/.config/fish/completions/shelf.fish

# powershell — add to your $PROFILE
shelf --completion powershell >> $PROFILE
```

## True junk

Some files aren't clutter, they're garbage — and shelf can take those out
for you. Mark a rule with `delete: true` and matching files go to shelf's
private trash, where they wait:

```yaml
keep_deleted: 30d

rules:
  - name: True junk
    match: ["*.tmp", "*.log"]
    older_than: 90d
    delete: true
```

While a file sits in trash, `shelf --undo` brings it right back. After
`keep_deleted` (30 days by default) passes, the next run purges it for
real. Delete rules *require* an `older_than`, so nothing young is ever
touched, and previews never purge — only `--apply` and `--watch` do.

## Is it safe?

This is the part we care about most:

- preview by default; moving requires the explicit `--apply`
- symlinks are followed nowhere, dotfiles invisible to it, directories sacred
- partial downloads excluded by extension
- watch mode waits about twenty seconds before judging a fresh file
- destination on another disk? the move falls back to a verified copy+delete
  that preserves timestamps and still refuses to overwrite anything
- "deleted" files rest in a private trash first — undoable for 30 days,
  and previews never purge anything
- and if anything ever feels wrong: `--undo`

Works the same on Linux, macOS and Windows.

## Where it's going

- [x] cross-device moves (copy+delete fallback when rename can't)
- [x] true junk: `delete` rules — trashed, journaled, undoable, auto-purged
- [x] shell completions for bash, zsh, fish and PowerShell (`--completion`)
- [ ] Homebrew / Scoop / AUR packages

Built by hand, for humans. If shelf saved you an hour of filing, consider
giving it a star — that's how other tired people find it.

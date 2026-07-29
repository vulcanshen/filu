# filu

<p align="center"><img src="docs/icon.svg" width="128" alt="filu icon" /></p>

[![GitHub Release](https://img.shields.io/github/v/release/vulcanshen/filu)](https://github.com/vulcanshen/filu/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vulcanshen/filu)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

**Language**: English · [繁體中文](README-zh_TW.md)

**A terminal file manager** — `Tab` / `Space` / `Enter` / `Esc` drive everything. No hotkey memorization, no setup, no learning curve. An information-rich file list, a marks bucket for copy/move, streaming file finders, live preview, and cd-on-quit are built in.

> _When in doubt, hit_ **`Space`**.

filu is a member of the `u`-family and a filesystem-domain implementation of [Vulcan's TUI Design Principle](https://github.com/vulcanshen/thoughts/blob/main/vtp.md) — the same design system as [kbu](https://github.com/vulcanshen/kbu). See [`docs/filu-implementation.md`](docs/filu-implementation.md).

## Demo

### Getting around filu
![basics](docs/demo-basics.gif)

### Marks — copy / move across tabs
![marks](docs/demo-marks.gif)

### Streaming finders — fuzzy name & ripgrep content
![finders](docs/demo-finders.gif)

### Favorites — star directories, manage them in the [3] Favorites tab
![favorites](docs/demo-favorites.gif)

### Preview, then yank to the clipboard
![preview](docs/demo-preview.gif)

### A shell in the active tab's directory
![shell](docs/demo-shell.gif)

## Five keys to drive filu

| Key | Behavior |
|---|---|
| **`Tab`** | Switch panel focus (or `1`–`3` directly) |
| **`Enter`** | Enter a directory / commit a choice |
| **`Space`** | *What can I do here?* — opens a contextual menu on every panel |
| **`Esc`** | Back out — up one directory / close any popup |
| **`?`** | Global help — every app-wide action in one list |

When in doubt, press `Space`. Power-user hotkeys (`o` open / `O` open-with / `m` mark / `c` copy / `v` move / `f` favorite / `y` yank / `r` rename / `a` add / `s` shell / `D` delete / `S` sort / `/` search / `go` goto / `b` breadcrumb / `z` zoom / …) exist for speed — every one is also reachable through the `Space` menu, so nothing's required to memorize unless you want it.

## Install

> filu is **macOS / Linux only** (no native Windows build; use WSL).

### Quick Install (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/install.sh | sh
```

Downloads the release binary, then fetches any missing `ripgrep` / `fd` into the same directory (from each tool's own GitHub release — no sudo, and named `rg`/`fd`, not Debian's `fdfind`). A platform with no prebuilt binary (e.g. `fd` on Intel macOS) falls back to an install hint.

### Homebrew (macOS/Linux)

```bash
brew install vulcanshen/tap/filu
```

The formula declares `ripgrep` + `fd` as dependencies, so content search and the finder listing work out of the box.

### From source

```bash
go install github.com/vulcanshen/filu/cmd/filu@latest
```

### Build locally

```bash
git clone https://github.com/vulcanshen/filu.git
cd filu
CGO_ENABLED=0 go build -o filu ./cmd/filu   # or: make build
./filu
```

A `Makefile` wraps the common tasks — `make build` (→ `./filu`), `make install`
(→ `$GOPATH/bin`, putting `filu` on your `PATH`) / `make uninstall`, `make package`
(a `.tar.gz` for your platform under `dist/`), and `make check` (fmt + vet + test).
Run `make` to list them.

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/uninstall.sh | sh
```

## Quick Start

```bash
filu
```

Opens on your current directory, focused on the file list. Press `Enter` to enter a directory, `Space` for the contextual menu, `Esc` to back out, `Tab` to move between panels.

To have filu change your shell's directory when you quit (the `q` picker), add one line to `~/.zshrc` / `~/.bashrc` and launch with **`filu`** (not `./filu`):

```sh
eval "$(filu shell)"
```

filu's cd-on-quit follows [superfile](https://github.com/yorukot/superfile)'s `cd_on_quit`, and its finders take after [LazyVim](https://github.com/LazyVim/LazyVim)'s search. Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

---

> The rest of this README is the operations manual — read on if you want the full feature surface, every keybinding, and configuration details.

## Features

- **Zero learning curve** — every action surfaces through the `Space` menu. Power-user hotkeys exist for speed but you can ignore the whole cheat sheet; `Space` walks you through the same menus, in context, on every panel. Onboarding doc: *"When in doubt, hit Space."*
- **3-panel workspace** — `[1]` the file list (main surface), `[2]` Preview, `[3]` a tabbed **Marks | Tasks | Favorites** panel. The list and preview share the top row at 2:1 (the info-rich list earns the width); the tabbed panel spans the full width below. `Tab` cycles the three (or `1`–`3` to jump); `h`/`l` switches a panel's own tabs.
- **Information-rich file list** — each row is a set of columns: a status glyph, `Modified`, `Owner` (user:group), `Perms` (eza-coloured `r` yellow / `w` red / `x` green), `Size` (eza colour-scale, warmer as it grows; directories show `-`, never a recursive sum), and the icon + name. The column header doubles as the sort indicator, and columns drop gracefully (owner → size → modified → perms) as the panel narrows, the name always last to go.
- **Per-directory sort** — `S` picks a column (Name / Modified / Owner / Perms / Size) and direction, building a multi-tier chain. Each directory remembers its own sort — set one for `~/Downloads` and it sticks there, independent of every other directory — persisted in `state.yaml`.
- **Powerline breadcrumb header** — the active tab's full path renders as a powerline breadcrumb across the top, coloured along a `crust → blue` gradient by directory depth (root is the darkest chip, where you are is blue). Overflows shrink front segments to their initial (`~/Documents/x` → `~/D/x`), then collapse the middle to `…`.
- **Marks copy & move** — deferred cp/mv, like Finder's Cmd+C / Cmd+V. `m` marks a file into the marks bucket (a glyph marks it in the list — an in-place multi-select), then you navigate to the target directory and `c` copies / `v` moves the whole bucket there. In the Marks tab, `p` picks a *subset* to land instead of the whole bucket, and `m` unmarks an item. Copy leaves the bucket intact so you can land to several directories; move updates the bucket paths so they stay valid.
- **Async land with a readable Tasks tab** — copy/move runs in the background; progress streams into `[3]`'s Tasks tab as a plain-language, timestamped log (`2026-07-28 14:32:07  Copied report.pdf → proj` / `… Move 5 items → proj (2/5 failed)`). Same-disk moves are instant `rename`; cross-disk / copy shows progress. `D` drops a line from the log. Interrupted tasks are saved to `state.yaml` and restore as pending.
- **Native streaming finders** — a filu-drawn split picker (list + preview), not the fzf binary. Every mode streams its listing (`fd`-order, first results near-instant, filter while it loads):
  - **`/` Search** — opens a chooser: **filename** (fuzzy-match names in the current tab's subtree; a query starting with `/` re-anchors onto an absolute path instead, so anywhere on disk is reachable) or **content** (filter by `ripgrep`; the preview scrolls to the matched line and highlights it).
  - **`go` Goto** — a picker for a **Favorite** directory, or a fuzzy **Search** of directory paths under `$HOME`; a query starting with `/` re-anchors onto that absolute path — fuzzy across the whole path, bounded a few levels below the deepest directory it names — so directories outside home are reachable too. `Enter` teleports the active tab there.
  - **`b` Breadcrumb** — a popup of the current tab's ancestor directories; `Enter` jumps the tab up to any level.
  - In the result list, `Esc` leaves and `q` returns to the input. Scan bounds (`finder_cap`) and skipped tool directories (`ignore_dirs`) are tunable in `config.yaml`.
- **Favorites** — `f` favorites the cursor directory (a star marks it in the list). Panel `[3]`'s **Favorites** tab lists them by full path — `D` removes one right there; to jump to a favorite, use **Goto → Favorites**. Browser-bookmark semantics — a saved place you jump back to.
- **Open, and open-with** — `o` opens the cursor file or directory with the OS default app. `O` (shift-o) opens a picker instead: **Default** plus the apps you list under `open_with` in `config.yaml` (VSCode, IntelliJ IDEA, …), each run as `<cmd> <path>` — handy for opening the current folder in your IDE. In filu, `o` / `O` — not `Enter` — open things; `Enter` is navigation only.
- **Drop to a shell** — `s` confirms the target directory, then opens your `$SHELL` in the active tab's directory inside the embedded terminal; run a few commands and `exit` to come back (the directory reloads in case files changed). It's a modal sub-shell — you leave it by exiting, not by switching away.
- **Preview by file kind** — detected from magic bytes: directory → inner tree, archive (zip / tar / tar.gz…) → contents, image → base64 `data:` URI, SVG → highlighted XML, text → syntax-highlighted with line numbers (Chroma / catppuccin-mocha), binary → hex + ASCII, PDF → extracted text + page count.
- **Yank with visual selection** — `y` on `[2]` Preview opens a viewport with a vim-style cursor; `v` enters character-wise visual selection, `y` copies the selection (or the whole content when nothing is selected) via OSC 52 (works through tmux / SSH). `y` on a file row or a Marks item copies its full path.
- **Delete to system trash** — `D` (with a confirmation dialog) moves to the OS trash (macOS Trash / Linux XDG). Restore via your file manager's trash UI.
- **Dynamic directory tabs** — `[1]` opens with one tab; `t` opens a new tab via a `{Same, Favorites, Search}` picker (up to five total), `w` closes the active one; reaching the limit toasts instead of silently doing nothing. Tabs are labelled with Roman numerals (`Ⅰ` … `Ⅴ`) — the path lives in the header, so the tab bar just marks position and which is active.
- **Launch-dir status bar** — under the header, a right-aligned line shows the directory filu was launched from (marked with a lavender glyph) — the fixed reference the cd-on-quit picker returns to.
- **eza icons + colours** — file-type glyphs come from eza's full icon table (~760 glyphs); colours come from a baked-in `vivid generate catppuccin-mocha` `LS_COLORS` palette resolved in eza's order (directory → symlink → executable → longest suffix → extension). No `LS_COLORS` needed at run time — every install shows the same palette, matching your terminal's `eza` / `ls`.
- **Live refresh** — the list tabs watch their directories (fsnotify) and reload when files change externally, keeping the cursor on its entry; bursts are debounced.
- **Session persistence** — extra tabs (dir + cursor), the marks bucket, favorites, tasks, and per-directory sorts are saved to `state.yaml`; the first tab always reopens at the launch directory, and launch always focuses the list.
- **cd-on-quit** — `q` opens a picker to leave your shell in the launch directory or any tab's directory (with the shell wrapper installed; see [cd-on-quit](#cd-on-quit)).
- **Vim-style navigation** — `j`/`k`, `u`/`d` half-page, `gg`/`G`, `h`/`l` switch the focused panel's tab.
- **Panel zoom** — `z` expands the focused panel full-screen; `z` again restores the grid.
- **CJK Nerd Font width** — a startup CPR probe measures the real icon cell width so panel borders don't break on CJK Nerd Fonts (e.g. Maple Mono NF CN) that paint file-type icons two cells wide.
- **unix-first, static binary** — macOS + Linux only (no native Windows build — `GOOS=windows` fails on purpose); `CGO_ENABLED=0` for a static binary.

## Key Bindings

### Primary interaction: five keys

| Key | Behavior |
|---|---|
| **`Tab`** | **Panel** — move focus to the next panel (or `1`–`3` to jump directly) |
| **`Enter`** | **Into** — enter a directory / commit a popup choice |
| **`Space`** | **Menu** — open the contextual menu wherever focus is. Also closes any open popup |
| **`Esc`** | **Back** — up one directory / close any popup |
| **`?`** | **Help** — the global (non-contextual) action list |

Where a contextual menu exists, `Space` is enough — you don't need to memorize the per-action keys. `h`/`l` switch the focused panel's tab (`[1]` directory tabs, `[3]` Marks / Tasks / Favorites).

### Accelerators — cursor + power triggers

```
 cursor    j k        u d         gg G        h l (switch this panel's tab)
 list      o open     O open-with  m mark     c copy    v move    f favorite
           y yank     r rename     a add      s shell   D delete  S sort   . hidden   z zoom
 finders   / search   go goto      b breadcrumb
 tabs      t new tab  w close tab
```

`gg` (jump to top) is a vim g-prefix chord — a lone `g` arms and waits; `go` (open the Goto picker) is the same prefix. `G` jumps to the bottom.

### Global

| Key | Action |
|---|---|
| `?` | Help popup |
| `q` | Quit — opens the cd-on-quit picker (leave the shell in a chosen directory) |
| `Ctrl+C` | Quit immediately (kills any running copy/move) |
| `y` | Copy the focused element's path / content to the clipboard (OSC 52) |

### Panel Space menus

| Focus | Menu items |
|---|---|
| **`[1]` List** | Open `o`, Open with `O`, Mark `m`, Yank `y`, Rename `r`, Delete `D`, Favorite `f` · Copy `c`, Move `v`, Search `/`, Goto `go`, Breadcrumb `b`, Tab `t`, Close tab `w`, Add `a`, Sort `S`, Shell `s`, Hidden `.`, Zoom `z` |
| **`[2]` Preview** | Yank `y`, Zoom `z` |
| **`[3]` Marks** | Pick `p`, Yank `y`, Unmark `m` · Switch tab `l`, Zoom `z` |
| **`[3]` Tasks** | Delete `D` · Switch tab `l`, Zoom `z` |
| **`[3]` Favorites** | Delete `D` (unfavorite) · Switch tab `l`, Zoom `z` |

## cd-on-quit

Pressing `q` opens a picker of distinct directories — the launch directory plus each tab's current directory — and, on exit, changes your shell's working directory to the one you pick (like superfile's `cd_on_quit`).

**Why one line of shell config is needed (it's an OS limit, not laziness):** a process can only change *its own* working directory — no syscall can change the *parent* process's (your shell's) cwd. filu is a child of your shell, so its own `cd` doesn't affect the shell. The only thing that can natively change the shell's cwd is a shell **builtin**, and filu is an external binary. So the standard fix is a two-part handshake: the program writes the chosen directory to a file, and a shell wrapper reads it and `cd`s. Add the wrapper to `~/.zshrc` / `~/.bashrc`:

```sh
eval "$(filu shell)"
```

Then launch with **`filu`** (not `./filu` — the wrapper is a shell function intercepting the command name `filu`; a path-qualified call bypasses it). filu still works without the wrapper; it just won't change your shell's directory on exit.

## Configuration

filu reads user settings from `config.yaml` in the OS config directory. `state.yaml` (auto-managed session state) sits alongside it; the two are kept separate on purpose — `config.yaml` is your hand-edited file, `state.yaml` is rewritten every quit.

| OS | Path |
|---|---|
| Linux | `$XDG_CONFIG_HOME/filu/` or `~/.config/filu/` |
| macOS | `~/Library/Application Support/filu/` |

A commented template is written on first launch (an existing file is never overwritten). The finder knobs (`finder_cap`, `ignore_dirs`) plus the `[o]pen` app list:

```yaml
# How many entries a finder scans before it stops. Goto walks all of $HOME,
# so this bounds it — raise it to reach more directories, lower it if the
# fuzzy filter lags on a large home.
finder_cap: 50000

# Directories the finders skip — caches, build output, IDE metadata, container
# data you never cd into. A bare name matches at any depth; a name with a slash
# (e.g. go/pkg) matches a path. Set to [] to exclude nothing.
ignore_dirs:
  - node_modules
  - .git
  - Library
  - OrbStack
  - go/pkg
  - vendor
  - target
  - __pycache__
  - .venv
  - .idea
  - .vscode
  - .cache
  - .Trash

# Apps for the [O]pen-with picker (press O on a file or directory; plain o just
# opens with the OS default). Each entry is a name + a command; filu runs
# `<cmd> <path>`. "Default" (the OS default app) is always offered first.
open_with:
  - name: VSCode
    cmd: code
  - name: IntelliJ IDEA
    cmd: idea
```

## Requirements

- **A Nerd Font** — filu uses Nerd Font glyphs (file-type icons, powerline chips) as visual vocabulary; it is part of the design, not optional. On a CJK Nerd Font (e.g. Maple Mono NF CN) that paints icons two cells wide, filu probes the real cell width at startup (CPR) and lays out to match, so borders stay aligned.
- **ripgrep** — required for `/` Search's content mode.
- **fd** — used to list files for the finders; falls back to a Go walk if absent.
- **macOS or Linux** — no native Windows build (`GOOS=windows` fails on purpose); use WSL.
- **Go 1.26+** — to build from source (`CGO_ENABLED=0` for a static binary).

## License

[GPL-3.0](LICENSE)

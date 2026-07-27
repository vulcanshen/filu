# filu

<p align="center"><img src="docs/icon.svg" width="128" alt="filu icon" /></p>

[![GitHub Release](https://img.shields.io/github/v/release/vulcanshen/filu)](https://github.com/vulcanshen/filu/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vulcanshen/filu)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

**Language**: English · [繁體中文](README-zh_TW.md)

**A single-pane terminal file manager** — `Tab` / `Space` / `Enter` / `Esc` drive everything. No hotkey memorization, no setup, no learning curve. Carry-bucket copy/move, streaming file finders, live preview, and cd-on-quit are built in.

> _When in doubt, hit_ **`Space`**.

filu is a member of the `u`-family and a filesystem-domain implementation of the same **ZLC** (Zero Learning Curve) design system as [kbu](https://github.com/vulcanshen/kbu) — see [`docs/zlc-filu-implementation.md`](docs/zlc-filu-implementation.md).

## Demo

> _Demo GIFs coming soon._

<!--
### Getting around filu
![basics](docs/demo-basics.gif)

### Carry-bucket copy & move
![carry](docs/demo-carry.gif)

### Streaming finders — name, content, goto
![finders](docs/demo-finders.gif)

### Preview & metadata side by side
![preview](docs/demo-preview.gif)
-->

## Five keys to drive filu

| Key | Behavior |
|---|---|
| **`Tab`** | Switch panel focus (or `1`–`5` directly) |
| **`Enter`** | Enter a directory / commit a choice |
| **`Space`** | *What can I do here?* — opens a contextual menu on every panel |
| **`Esc`** | Back out — up one directory / close any popup |
| **`?`** | Global help — every app-wide action in one list |

When in doubt, press `Space`. Power-user hotkeys (`o` open / `O` open-with / `p` pick / `y` yank / `c` copy / `m` move / `r` rename / `a` add / `s` shell / `D` delete / `P` pin / `/` search / `f` find / `go` goto / `b` breadcrumb / `z` zoom / …) exist for speed — every one is also reachable through the `Space` menu, so nothing's required to memorize unless you want it.

## Install

> [!NOTE]
> **Pre-release.** filu has no tagged release yet, so the release-binary channels
> below (`install.sh`, Homebrew, the release badge) go live only after the first
> `v*` tag. For now, **[build from source](#build-locally)** or `go install`.

> filu is **macOS / Linux only** (no native Windows build; use WSL).

### Quick Install (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/install.sh | sh
```

Downloads the release binary and prints install hints for `ripgrep` / `fd` if they're missing.

### Homebrew (macOS/Linux)

```bash
brew install vulcanshen/tap/filu
```

The cask declares `ripgrep` + `fd` as dependencies, so Find and the finder listing work out of the box.

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

Opens on your current directory. Press `Enter` to enter a directory, `Space` for the contextual menu, `Esc` to back out, `Tab` to move between panels.

To have filu change your shell's directory when you quit (the `q` picker), add one line to `~/.zshrc` / `~/.bashrc` and launch with **`filu`** (not `./filu`):

```sh
eval "$(filu shell)"
```

filu's cd-on-quit follows [superfile](https://github.com/yorukot/superfile)'s `cd_on_quit`, and its Search / Find / Goto finders take after [LazyVim](https://github.com/LazyVim/LazyVim)'s search. Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

---

> The rest of this README is the operations manual — read on if you want the full feature surface, every keybinding, and configuration details.

## Features

- **Zero learning curve** — every action surfaces through the `Space` menu. Power-user hotkeys exist for speed but you can ignore the whole cheat sheet; `Space` walks you through the same menus, in context, on every panel. Onboarding doc: *"When in doubt, hit Space."*
- **5-panel workspace** — `[1]` Places + Pinned, `[2]` the file list (main surface), `[3]` Preview, `[4]` Carries / Tasks, `[5]` file Metadata. Grid: `[1][2][3] / [1][2][3] / [4][4][5]` — the left column stacks list-over-carry, the right column mirrors it with preview-over-metadata. `Tab` cycles all five (or `1`–`5` to jump).
- **Powerline breadcrumb header** — the active tab's full path renders as a powerline breadcrumb across the top, coloured along a `crust → blue` gradient by directory depth (root is the darkest chip, where you are is blue). Overflows shrink front segments to their initial (`~/Documents/x` → `~/D/x`), then collapse the middle to `…`. The tab-numeral + folder glyph lead the bar.
- **Directory status bar** — under the header, an eza-coloured status line for the current directory: permissions (`r` yellow / `w` red / `x` green), owner:group, item + hidden counts, and free / total disk. Everything is computed once when the directory loads (one `stat` + one `statfs`, never a recursive size walk) so it stays instant.
- **Carry-bucket copy & move** — deferred cp/mv, like Finder's Cmd+C / Cmd+V. `p` picks a file into the carries bucket (a green check-circle marks it in the list — an in-place multi-select), then you navigate to the target directory and `c` copies / `m` moves the whole bucket there. In the Carries tab (`[4]`), `p` picks a *subset* to land instead of the whole bucket (marked with its own distinct glyph). Copy leaves the bucket intact so you can land to several directories; move updates the bucket paths so they stay valid.
- **Async land with a Tasks tab** — copy/move runs in the background; progress streams into `[4]`'s Tasks tab (running / done / pending / error). Same-disk moves are instant `rename`; cross-disk / copy shows progress. Interrupted tasks are saved to `state.yaml` and can be re-run with `R` next launch.
- **Native streaming finders** — a filu-drawn split picker (list + preview), not the fzf binary. Four modes, all with a right-side preview, all streaming their listing (`fd`-order, first results near-instant, filter while it loads):
  - **`/` Search** — fuzzy-match file *names* in the current tab's subtree.
  - **`f` Find** — filter by *content* via `ripgrep`; the preview scrolls to the matched line and highlights it.
  - **`go` Goto** — fuzzy-match *directory paths* under `$HOME` (directories only); `Enter` teleports the active tab there.
  - **`b` Breadcrumb** — a popup of the current tab's ancestor directories; `Enter` jumps the tab up to any level.
  - In the result list, `Esc` leaves and `q` returns to the input. Scan bounds (`finder_cap`) and skipped tool directories (`ignore_dirs`) are tunable in `config.yaml`.
- **Open, and open-with** — `o` opens the cursor file or directory with the OS default app. `O` (shift-o) opens a picker instead: **Default** plus the apps you list under `open_with` in `config.yaml` (VSCode, IntelliJ IDEA, …), each run as `<cmd> <path>` — handy for opening the current folder in your IDE. In filu, `o` / `O` — not `Enter` — open things; `Enter` is navigation only.
- **Drop to a shell** — `s` opens your `$SHELL` in the active tab's directory inside the embedded terminal; run a few commands and `exit` to come back (the directory reloads in case files changed). It's a modal sub-shell — you leave it by exiting, not by switching away — so quick per-tab work never needs you to quit filu.
- **Preview by file kind** — detected from magic bytes: directory → inner tree, archive (zip / tar / tar.gz…) → contents, image → base64 `data:` URI, SVG → highlighted XML, text → syntax-highlighted with line numbers (Chroma / catppuccin-mocha), binary → hex + ASCII, PDF → extracted text + page count.
- **File metadata panel** — `[5]` shows `stat`-level metadata for the cursor file (Name / Path / Type / Size / Owner / Group / Links / Inode / Perm / Octal / Modified / Accessed / Changed / Created).
- **Yank with visual selection** — `y` on `[3]` Preview or `[5]` Meta opens a viewport with a vim-style cursor; `v` enters character-wise visual selection, `y` copies the selection (or the whole content when nothing is selected) via OSC 52 (works through tmux / SSH). `y` on a file row or a Carries item copies its full path.
- **Delete to system trash** — `D` (with a confirmation dialog) moves to the OS trash (macOS Trash / Linux XDG). Restore via your file manager's trash UI.
- **Dynamic directory tabs** — `[2]` opens with one tab; `t` opens a new tab in the current directory, `T` opens one at a directory you pick via Goto (up to five total), `w` closes the active one; reaching the limit toasts instead of silently doing nothing. Tabs are labelled with Roman numerals (`Ⅰ` … `Ⅴ`) — the path lives in the header, so the tab bar just marks position and which is active.
- **eza icons + colours** — file-type glyphs come from eza's full icon table (~760 glyphs); colours come from a baked-in `vivid generate catppuccin-mocha` `LS_COLORS` palette resolved in eza's order (directory → symlink → executable → longest suffix → extension). No `LS_COLORS` needed at run time — every install shows the same palette, matching your terminal's `eza` / `ls`.
- **Live refresh** — the list tabs watch their directories (fsnotify) and reload when files change externally, keeping the cursor on its entry; bursts are debounced.
- **Session persistence** — extra tabs (dir + cursor), focus, carry bucket, pinned dirs, tasks, and sort are saved to `state.yaml`; the first tab always reopens at the launch directory.
- **cd-on-quit** — `q` opens a picker to leave your shell in the launch directory or any tab's directory (with the shell wrapper installed; see [cd-on-quit](#cd-on-quit)).
- **Vim-style navigation** — `j`/`k`, `u`/`d` half-page, `gg`/`G`, `h`/`l` switch the focused panel's tab.
- **Pinned places** — `[1]` has system Places (LaunchDir / Home / Root) plus a Pinned section; `P` on a directory pins it, and pinned paths shorten the same progressive way the header does.
- **Panel zoom** — `z` expands the focused panel full-screen; panels with tabs lay them out as equal columns (by the actual tab count). `z` again restores the grid.
- **CJK Nerd Font width** — a startup CPR probe measures the real icon cell width so panel borders don't break on CJK Nerd Fonts (e.g. Maple Mono NF CN) that paint file-type icons two cells wide.
- **unix-first, static binary** — macOS + Linux only (no native Windows build — `GOOS=windows` fails on purpose); `CGO_ENABLED=0` for a static binary.

## Key Bindings

### Primary interaction: five keys

| Key | Behavior |
|---|---|
| **`Tab`** | **Panel** — move focus to the next panel (or `1`–`5` to jump directly) |
| **`Enter`** | **Into** — enter a directory / commit a popup choice |
| **`Space`** | **Menu** — open the contextual menu wherever focus is. Also closes any open popup |
| **`Esc`** | **Back** — up one directory / close any popup |
| **`?`** | **Help** — the global (non-contextual) action list |

Where a contextual menu exists, `Space` is enough — you don't need to memorize the per-action keys. `h`/`l` switch the focused panel's tab (`[2]` directory tabs, `[4]` Carries / Tasks).

### Accelerators — cursor + power triggers

```
 cursor    j k        u d        gg G        h l (switch this panel's tab)
 panel 2   o open     O open-with  p pick    y yank    c copy   m move
           r rename   a add        s shell   D delete  P pin    . hidden  z zoom
 finders   / search   f find     go goto     b breadcrumb
 tabs      t new tab  T new tab @ goto  w close tab
```

`gg` (jump to top) is a vim g-prefix chord — a lone `g` arms and waits; `go` (open the Goto finder) is the same prefix. `G` jumps to the bottom.

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
| **`[1]` Places** | Jump (`Enter`), UnPin (`P`, pinned rows) |
| **`[2]` List** | Open `o`, Open with `O`, Pick `p`, Yank `y`, Rename `r`, Delete `D`, Pin `P` · Copy `c`, Move `m`, Search `/`, Find `f`, Goto `go`, Breadcrumb `b`, Tab `t`, Tab @ goto `T`, Close tab `w`, Add `a`, Shell `s`, Hidden `.`, Zoom `z` |
| **`[3]` Preview** | Yank `y`, Zoom `z` |
| **`[4]` Carries** | Pick `p`, Yank `y`, Delete `D` · Tab `l`, Zoom `z` |
| **`[4]` Tasks** | Redo `R`, Delete `D` · Tab `l`, Zoom `z` |
| **`[5]` Meta** | Yank `y`, Zoom `z` |

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
- **ripgrep** — required for `f` Find (content search).
- **fd** — used to list files for the finders; falls back to a Go walk if absent.
- **macOS or Linux** — no native Windows build (`GOOS=windows` fails on purpose); use WSL.
- **Go 1.26+** — to build from source (`CGO_ENABLED=0` for a static binary).

## License

[GPL-3.0](LICENSE)

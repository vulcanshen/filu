# filu

<p align="center"><img src="docs/icon.svg" width="128" alt="filu icon" /></p>

[![GitHub Release](https://img.shields.io/github/v/release/vulcanshen/filu)](https://github.com/vulcanshen/filu/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vulcanshen/filu)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

**Language**: English · [繁體中文](README-zh_TW.md)

**A terminal file manager you don't have to learn.** Four keys — `Tab`, `Enter`, `Space`, `Esc` — reach everything filu can do. Browse a file list that shows what matters at a glance, gather files and drop them somewhere else, find anything by name or content, preview as you go, and leave your shell in the directory you ended up in.

> _When in doubt, hit_ **`Space`**.

## Demo

![demo](docs/demo-basics.gif)

Getting around filu: panels, tabs, the `Space` menu and the path you can always see.

## Why filu

- **Nothing to memorize.** Press `Space` on any panel and filu lists what you can do there. Every hotkey is a shortcut to a menu item — never the only way in.
- **A directory at a glance.** Modified time, owner, permissions and size sit beside every name, coloured the way your `eza` / `ls` colours them. Sort by any column, and each directory remembers its own sort.
- **Copy and move like a desktop.** Mark files as you browse, walk to the destination, press `c` or `v`. Mark across tabs, land the same files in several places, or zip them up first.
- **Find it fast.** Fuzzy-search file names, search file contents with ripgrep, or jump to any directory under your home — results stream in as you type, with a live preview beside them.
- **Look before you open.** Syntax-highlighted text, archive contents, directory trees, PDFs, images, hex for binaries. Select part of it and copy it to your clipboard — even over SSH.
- **Leave where you're working.** Quit with `q` and your shell lands in whichever directory you pick.
- **Picks up where you left off.** Tabs, marks, favorites and per-directory sorts survive a restart.

## Install

### Requirements

- **A Nerd Font** in your terminal — filu's icons are Nerd Font glyphs. CJK Nerd Fonts (e.g. Maple Mono NF CN) work too.
- **A truecolor (24-bit) terminal** — filu's colours and the dimming under popups need it; with 256 colours the shades blur together.
- **ripgrep** for content search, and **fd** for fast finders (the quick installer and Homebrew handle both).
- **macOS or Linux** — or WSL on Windows.

### Quick install

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/install.sh | sh
```

Installs filu, plus `ripgrep` and `fd` if you don't have them — no sudo needed.

### Homebrew

```bash
brew install vulcanshen/tap/filu
```

`ripgrep` and `fd` come along as dependencies.

### Go

```bash
go install github.com/vulcanshen/filu/cmd/filu@latest
```

Needs Go 1.26+. Install `ripgrep` and `fd` yourself for content search and fast finders.

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/uninstall.sh | sh
```

## Getting started

```bash
filu              # open the current directory
filu ~/proj       # open a directory
filu ~/notes.md   # open its directory, with the cursor on that file
```

filu opens with the file list focused. Five keys cover the rest:

| Key | What it does |
|---|---|
| **`Tab`** | Move to the next panel (`Shift-Tab` goes back, or press `1`–`3` to jump) |
| **`Enter`** | Go into a directory, see a file's details, or confirm a choice |
| **`Space`** | *What can I do here?* — the menu for wherever you are |
| **`Esc`** | Back out — up one directory, or close a popup |
| **`?`** | Keys — what you can press right here (in a popup, that popup's keys) |

### Let `q` change your shell's directory

Add this line to `~/.zshrc` or `~/.bashrc`:

```sh
eval "$(filu shell)"
```

Then start filu as **`filu`** (not `./filu`). Pressing `q` opens a picker of the directory you launched from plus every tab's directory; pick one and your shell is there when filu closes. Without the line, filu works the same — quitting just leaves your shell where it was.

## The screen

Three panels:

- **`[1]` Files** — the main list, with a path row on top. Up to five tabs, each browsing its own directory.
- **`[2]` Preview** — whatever the cursor is on.
- **`[3]` Marks | Tasks | Favorites** — the files you've gathered, the log of copies and moves, and your saved directories.

`Tab` (or `1`–`3`) moves between panels, `h` / `l` switches a panel's tabs, and `z` zooms the focused panel to full screen.

## What you can do

### Browse

- `Enter` goes into a directory — on a file it shows the details: full path, type, size, times, permissions, owner. `Esc` goes up; `j` / `k`, `u` / `d` for half a page, `gg` / `G` for top and bottom.
- `b` jumps up to any parent directory. `.` shows or hides hidden files.
- `S` sorts by name, modified time, owner, permissions or size — stack several for tie-breaks. The sort sticks to that directory only.
- `t` opens a new tab (same directory, a favorite, or a search), `w` closes it.
- Files changed by other programs show up on their own.

### Gather, copy, move

- `m` marks a file. Marks stay put while you move around, so you can gather from several directories and tabs.
- Go where the files belong and press `c` to copy or `v` to move them there. Copying keeps your marks, so you can land them in more than one place.
- In the **Marks** tab: `Enter` shows the file in `[1]` (in the tab already at its directory, or a new one), `p` picks just some of them to land, `m` unmarks one, `C` clears them all, and `Z` packs your picks into a zip that you then land with `c` / `v`.
- Copies and moves run in the background; the **Tasks** tab shows a plain-language log; `Enter` on an entry takes you to where it landed. A task cut short by quitting comes back next time.

### Find

- **`/` Search** — by **filename** (fuzzy, anywhere below the current directory) or by **content** (ripgrep; the preview jumps to the matching line). Start the query with `/` or `~/` to search from that path instead, anywhere on disk.
- **`go` Goto** — jump to a favorite, or fuzzy-search every directory under your home (hidden ones too). `Enter` takes the tab there.
- Results appear as they're found — start typing right away. While you type, `↑` / `↓` move among the results and `Enter` opens the highlighted one; `Tab` moves into the list, where `j` / `k` move too.

### Favorites

- `f` stars the directory under the cursor; `F` stars the one you're in. Starred directories are marked in the list.
- The **Favorites** tab lists them: `Enter` goes there (its tab, or a new one), `o` lets you choose the tab, `D` removes it. Goto → Favorites jumps there too.

### Preview and copy

- The preview shows text with syntax highlighting and line numbers, directory trees, archive contents, PDFs, images, SVG source, and hex for binaries.
- `y` (or `Enter`) in the preview opens it in a scrollable view: move with `h`/`j`/`k`/`l`, by word with `w`/`b`/`e` and to the line ends with `0`/`$`; `v` starts a selection, `y` copies it (or everything, if nothing is selected). While selecting, the box turns yellow with **Selection** at its top right, and `?` lists the selection keys.
- `y` on a file copies its full path. Copying works through tmux and SSH.

### Open, edit, and everything else

- `o` opens a file or directory with its default app. `O` lets you choose the app — add your own (VSCode, IntelliJ IDEA, …) in the config.
- `s` drops you into your shell in the current directory; type `exit` to come back, or press `Alt-Esc`: filu asks first, then `Enter` ends the shell (whatever is running in it) and `Esc` takes you back to it.
- `r` renames, `a` creates a file (end the name with `/` for a directory), `D` moves to the trash after asking. A name that is empty or already taken is refused on the spot — the box stays open and says why.

## Key reference

```
 cursor    j/k        u/d         gg/G        h/l (switch this panel's tab)
 list      o open     O open-with  m mark     c copy    v move    f favorite
           y yank     r rename     a add      s shell   D delete  S sort   . hidden   z zoom
 finders   / search   go goto      b breadcrumb
 tabs      t new tab  w close tab
```

| Key | Anywhere |
|---|---|
| `?` | The keys you can press here — dimmed when they can't do anything right now; `?` or `Esc` closes the list |
| `q` | Quit, choosing where your shell ends up (while typing, `q` is just a letter) |
| `Ctrl-C` | Same as `q`, even while typing; press it again on the quit picker to leave at once |

Every panel's `Space` menu lists what you can do to the item under the cursor, then to the panel, and ends with **Global operation**, which opens a menu of app-wide actions (for now: Quit `q`). A row that can't run right now — say, Close tab with only one tab open — is shown dimmed.

| Focus | Menu items |
|---|---|
| **`[1]` Files** | Open `o`, Open with `O`, Mark `m`, Yank `y`, Rename `r`, Delete `D`, Favorite `f` · Copy `c`, Move `v`, Search `/`, Goto `go`, Favorite dir `F`, Breadcrumb `b`, Switch tab `l`, Tab `t`, Close tab `w`, Add `a`, Sort `S`, Shell `s`, Hidden `.`, Zoom `z` |
| **`[2]` Preview** | Yank `y`, Zoom `z` |
| **`[3]` Marks** | Pick `p`, Yank `y`, Unmark `m` · Zip `Z`, Clear `C`, Switch tab `l`, Zoom `z` |
| **`[3]` Tasks** | Delete `D` · Switch tab `l`, Zoom `z` |
| **`[3]` Favorites** | Open in `o`, Delete `D` · Switch tab `l`, Zoom `z` |

## Configuration

Your settings live in `config.yaml`:

| OS | Directory |
|---|---|
| Linux | `$XDG_CONFIG_HOME/filu/` or `~/.config/filu/` |
| macOS | `~/Library/Application Support/filu/` (or `$XDG_CONFIG_HOME/filu/` when set) |

filu writes a commented copy the first time it runs and never overwrites yours. Next to it, `state.yaml` holds your session — leave that one to filu.

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

# Apps for the [O]pen-with picker (press [O] on a file or directory; plain [o] just
# opens with the OS default). Each entry is a name + a command; filu runs
# `<cmd> <path>`. "Default" (the OS default app) is always offered first.
open_with:
  - name: VSCode
    cmd: code
  - name: IntelliJ IDEA
    cmd: idea
```

## Limits

Not there, on purpose:
- **native Windows** — filu runs on macOS and Linux; on Windows, run it inside WSL
- **the mouse** — everything is on the keyboard
- **directory sizes** — a directory's size shows as `-`; filu never adds up a whole tree

## Links

- [CHANGELOG.md](CHANGELOG.md) — what each release changed
- [`docs/dev-remarks.md`](docs/dev-remarks.md) — the developer's notes: how it works, why, building from source, releasing

## terminu family

filu follows the [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.22/principle): the same keys and the same menus as the rest of the family — [kbu](https://github.com/vulcanshen/kbu) (Kubernetes), [sshu](https://github.com/vulcanshen/sshu) (ssh), [webu](https://github.com/vulcanshen/webu) (the web) and [locku](https://github.com/vulcanshen/locku) (screen lock).

## License

[GPL-3.0](LICENSE)

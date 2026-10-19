# filu

<p align="center"><img src="docs/icon.svg" width="128" alt="filu icon" /></p>

[![GitHub Release](https://img.shields.io/github/v/release/vulcanshen/filu)](https://github.com/vulcanshen/filu/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vulcanshen/filu)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

**Language**: English · [繁體中文](README-zh_TW.md)

**A terminal file manager you don't have to learn.** `Space` lists what you can do wherever you are, `?` lists the keys. Browse a file list that shows what matters at a glance, gather files and drop them somewhere else, find anything by name or content, preview as you go, and leave your shell in the directory you ended up in.

> _When in doubt, hit_ **`Space`**.

## Demo

![demo](docs/demo-basics.gif)

## Why filu

- **Nothing to memorize.** Press `Space` on any panel and filu lists what you can do there. Every hotkey is a shortcut to a row of that menu — never the only way in.
- **A directory at a glance.** Modified time, owner, permissions and size sit beside every name, coloured by type the way eza colours them (catppuccin-mocha). Sort by any column, stack several, and each directory remembers its own sort.
- **Copy and move like a desktop.** Mark files as you browse, walk to the destination, press `c` or `v`. Mark across directories and tabs, land the same files in several places, or zip them up first.
- **Find it fast.** Search file names, search file contents with ripgrep, or jump to any directory under your home — results appear as they are found, with a preview beside them.
- **Look before you open.** Syntax-highlighted text, archive contents, directory trees, PDFs (their text), images (as a data: URI), hex for binaries. Select part of it and copy it to your clipboard — even over SSH.
- **Leave where you're working.** Quit and choose where your shell lands: the directory you started in, or any tab's.
- **Picks up where you left off.** Tabs, marks, favorites, tasks and per-directory sorts survive a restart.

## Install

### Requirements

- **A Nerd Font** in your terminal — filu's icons are Nerd Font glyphs.
  filu asks the terminal at startup how many cells an icon takes, and a powerline symbol (the ends of the panel labels, the arrow at the end of the cursor). If borders still come out crooked, `filu iconwidth` shows what it measured, and you can set them yourself: `FILU__ICON_WIDTH=1` or `2` for icons, `FILU__POWERLINE_WIDTH=1` or `2` for powerline symbols. `TERMINU__ICON_WIDTH` and `TERMINU__POWERLINE_WIDTH` are the same settings shared by the whole terminu family: set them once and every family app reads them (the `FILU__` ones still win). Inside a family app's terminal — kbu's Alterm, filu's own shell — the outer app sets both for you.
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

### The first time

filu writes a commented `config.yaml` into its settings directory (see [Configuration](#configuration)) and opens on the directory you started it in. Nothing else is set up.

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/uninstall.sh | sh
```

It asks before removing your settings.

## Getting started

```bash
filu              # open the current directory
filu ~/proj       # open a directory
filu ~/notes.md   # open its directory, with the cursor on that file
filu help         # the command line's usage
```

filu opens with the file list focused. These keys cover the rest:

| Key | What it does |
|---|---|
| **`Space`** | *What can I do here?* — the menu for the panel you are on; its last row opens the app-wide actions |
| **`?`** | The keys you can press right here (in a popup, that popup's keys) |
| **`Enter`** | Go into a directory, see a file's details, or do what the row you are on is for |
| **`Esc`** | Back out: close the popup on top, or go up a directory |
| **`Tab`** | The next panel (`Shift-Tab` the one before, `1`–`3` jump to one) |
| **`q`** | Quit (it asks first — with the line below, it asks where your shell goes) |

### Let quitting change your shell's directory

Add this line to `~/.zshrc` or `~/.bashrc`:

```sh
eval "$(filu shell)"
```

Then start filu as **`filu`** (not `./filu`). `q` then asks where your shell goes: the directory you started in, or any tab's — press its number, or move to it and press `Enter`. While a copy or a move is still running, filu asks once more, since quitting stops it. Without the line, filu works the same — quitting just asks first and leaves your shell where it was.

## Usage

### The screen

Three panels:

- **`[1]` Files** — the list, with the directory's path on top. Up to five tabs, each on its own directory.
- **`[2]` Preview** — whatever the cursor in `[1]` is on.
- **`[3]` Marks | Tasks | Favorites** — the files you've gathered, the log of copies, moves and zips, and your starred directories.

`Tab` (or `1`–`3`) moves between panels, `h|l` switches a panel's tabs, and `z` zooms the panel you are on to the whole screen — zoomed, `[1]` shows all its tabs side by side.

### What you can do

#### Browse

- `Enter` goes into a directory; on a file it shows the details: full path, type, size, times, permissions, owner. `Esc` goes up; `j|k`, `u|d` for half a page, `gg|G` for top and bottom.
- `b` jumps up to any of the directories this one is in. `.` shows or hides hidden files.
- `S` sorts by name, modified time, permissions, owner or size — choose a column, then a direction. The columns stay open, showing the sort so far, so you can add a tie-break right away; `Esc` closes them. Directories stay first. The sort sticks to that directory only; `[r]eset` goes back to names.
- As the panel narrows, whole columns make way for the names — the owner first, then the size, the modified time and the permissions.
- `t` opens a new tab (the same directory, a favorite, or a search), `w` closes the one you are on.
- Files changed by other programs show up on their own.

#### Gather, copy, move

- `m` marks a file. Marks stay put while you move around, so you can gather from several directories and tabs.
- Go where the files belong and press `c` to copy or `v` to move them there. Copying keeps your marks, so you can land them in more than one place.
- In the **Marks** tab: `Enter` shows the file in `[1]` (in the tab already on its directory, or a new one) and takes you there, `p` picks some of them so that only the picks land, `m` unmarks one, `C` clears them all, `y` copies a path, and `Z` packs what would land — the picks, or every mark — into a zip that you then land with `c` or `v`. filu suggests the zip's name in grey: `Enter` takes it, or type your own.
- Copies, moves and zips run in the background; the **Tasks** tab logs them in plain words, with a spinner and a count on the ones still going, and `Enter` on one takes you to where it landed. A task cut short by quitting comes back marked as interrupted.

#### Find

- **`[/] Search`** — `[f]ile names`: everything under the directory, narrowed as you type (every word you type must appear, in any order). `[c]ontents` (ripgrep): the files that hold what you type, searched again as you type, each previewed from its first match.
- **`[go]to`** — jump to a favorite, or search every directory under your home (hidden ones too).
- In File names and Goto, type a path starting with `/` or `~/` to search from there instead: what is under the deepest directory it names, a few levels down, narrowed by the rest of what you typed — anywhere on the disk (`/usr/lo` finds `/usr/local`; `~/proj/` lists `~/proj`).
- Results appear as they're found — start typing right away. While you type, `↑|↓` move among the results; `Enter` or `Tab` goes to the list, where `j|k` move and `Enter` opens the one you are on; `Tab` or `/` goes back to typing; `Esc` closes. Each result is previewed beside the list, with its type, size, modified time and mode.

#### Favorites

- `f` stars the directory under the cursor; `F` stars the one you're in. A star shows after a starred directory's name, and after the path when you are in one.
- The **Favorites** tab lists them: `Enter` goes there (its tab, or a new one), `o` lets you choose the tab, `D` takes the star off (it asks first). `[go]to` → `[f]avorite` jumps there too.

#### Preview and copy

- The preview shows text with syntax highlighting and line numbers, directory trees, archive contents, PDFs (their text), images (as a data: URI), SVG source, and hex for binaries.
- In `[2]`, `Enter` opens the preview in a scrollable view: `v` brings out a cursor, `v` again starts selecting, move with `h|j|k|l`, by word with `w|b|e` and to the line ends with `0|$`, and `y` copies the selection. Without the cursor `y` copies all of it — as `y` on `[2]` itself does.
- `y` on a file in `[1]` copies its full path. Copying works through tmux and SSH.

#### Open, edit, and everything else

- `o` opens a file or directory with its default app (it asks first). `O` lets you choose the app — add your own (VSCode, IntelliJ IDEA, …) in the config.
- `s` drops you into your shell in the current directory (it asks first); type `exit` to come back, or press `Alt-Esc`: filu asks, then `Enter` ends the shell (whatever is running in it) and `Esc` takes you back to it.
- `r` renames (the old name shows in grey: `Tab` takes it to edit, or just type a new one), `a` creates a file (end the name with `/` for a directory), `D` moves to the trash after asking. A name that is empty, already taken, outside the current directory or holding a control character is refused on the spot — the box stays open and says why. If renaming or creating fails all the same (say, no permission), filu says why over the box, and closing that brings you back to what you typed.

### Key reference

| Key | `[1]` Files |
|---|---|
| `Enter` | into a directory · a file's details |
| `o` · `O` | open · open with |
| `m` · `c` · `v` | mark · copy the marks here · move them here |
| `y` · `r` · `D` | copy the path · rename · delete |
| `f` · `F` | star the directory under the cursor · the one you are in |
| `/` · `go` · `b` | search · goto · breadcrumb |
| `t` · `w` · `h\|l` | new tab · close tab · switch tab |
| `a` · `S` · `s` · `.` · `z` | add · sort · shell · hidden files · zoom |

| Key | `[2]` Preview | `[3]` Marks | `[3]` Tasks | `[3]` Favorites |
|---|---|---|---|---|
| `Enter` | view | show in `[1]` | go to where it landed | go there |
| others | `y` copy all · `z` zoom | `p` pick · `m` unmark · `y` copy the path · `Z` zip · `C` clear · `z` zoom | `D` take out of the log · `z` zoom | `o` open in · `D` take the star off · `z` zoom |

| Key | Anywhere |
|---|---|
| `?` | The keys you can press here; `?` or `Esc` closes the list |
| `q` · `Ctrl-C` | Quit (asks first, or asks where your shell goes); `Ctrl-C` works while typing too, and pressed again it leaves at once |

Every panel's `Space` menu lists what you can do to the item under the cursor, then to the panel, and ends with **Global operation**: the app-wide actions (`[q]uit`). A row that can't run right now — say, `[w] Close tab` with only one tab open — is shown dimmed.

## Configuration

Your settings live in `config.yaml`, in `$XDG_CONFIG_HOME/filu/` (or `~/.config/filu/`); `FILU__CONFIG` names another directory. filu writes a commented copy the first time it runs and never overwrites yours. Next to it, `state.yaml` holds where you were — the tabs, the marks, each directory's sort — leave that one to filu (`FILU__STATE` puts it in another directory). Your favorites and the task log are in `~/.filu/data.yaml` (`FILU__DATA` names another directory): that is the one to back up.

On macOS, versions up to 0.4.0 kept everything in `~/Library/Application Support/filu/`. The first time this version starts, it moves what is in there to `~/.config/filu/` and says so, and removes the emptied old directory. A file `~/.config/filu/` already has is left alone in both places, and filu shows you both paths so you can put the two together yourself. If filu cannot move them, it keeps using the old directory and shows you both paths, to move it yourself. Favorites and the task log move out of `state.yaml` into `~/.filu/data.yaml` the same way: if `data.yaml` already has its own, the ones in `state.yaml` stay there, and filu tells you.

```yaml
# How many entries a finder scans before it stops. Goto walks all of $HOME,
# so this bounds it.
finder_cap: 50000

# Directories the finders skip — caches, build output, IDE metadata, container
# data you never cd into. A bare name matches at any depth; a name with a slash
# (e.g. go/pkg) matches a path. Set to [] to exclude nothing.
ignore_dirs:
  - node_modules
  - .git
  - Library
  - go/pkg
  # …

# Apps for the Open with picker ([O] on a file or directory; [o] opens with the
# system's default). filu runs `<cmd> <path>`.
open_with:
  - name: VSCode
    cmd: code
  - name: IntelliJ IDEA
    cmd: idea

# How many cells a tab takes in the preview and the viewer.
input_tab_width: 4

# true keeps private keys, and files only their owner may read, out of the
# finders' previews.
hide_secrets_in_preview: false
```

`input_tab_width` and `hide_secrets_in_preview` are the same settings in every terminu family app.

## Limits

Not there, on purpose:
- **native Windows** — filu runs on macOS and Linux; on Windows, run it inside WSL
- **the mouse** — everything is on the keyboard
- **directory sizes** — a directory's size shows as `-`; filu never adds up a whole tree
- **pictures** — an image shows as its data: URI, not drawn
- **a terminal smaller than 60×24** — filu shows only how big a terminal it needs; make the window larger and everything is back as it was

## Links

- [CHANGELOG.md](CHANGELOG.md) — what each release changed
- [`docs/dev-remarks.md`](docs/dev-remarks.md) — the developer's notes: how it works, why, building from source, releasing

## terminu family

filu follows the [terminu design](https://github.com/vulcanshen/terminu) and is built on [terminu-go](https://github.com/vulcanshen/terminu-go): the same keys, menus and popups as the rest of the family — [kbu](https://github.com/vulcanshen/kbu) (Kubernetes), [sshu](https://github.com/vulcanshen/sshu) (ssh), [webu](https://github.com/vulcanshen/webu) (the web), [locku](https://github.com/vulcanshen/locku) (screen lock), [kamonu](https://github.com/vulcanshen/kamonu) (the family's marks) and [ccdrawu](https://github.com/vulcanshen/ccdrawu) (a sidebar for Claude Code).

## License

[GPL-3.0](LICENSE)

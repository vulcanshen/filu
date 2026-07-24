# Changelog

## [Unreleased]

### Added
- File-type icons now come from eza's full icon table (~760 glyphs, generated
  from eza's source). Every row in panel [2], the rename popup, and the Search
  list shows its eza glyph — languages, config files, archives, media, and
  special names (`README`, `Dockerfile`, `go.mod`, …). Directory and filename
  matches are case-sensitive; the extension match is lower-cased.
- Release tooling: a goreleaser config (linux/darwin, no Windows), an
  `install.sh` / `uninstall.sh`, and a tag-triggered GitHub Actions workflow. The
  Homebrew cask declares `ripgrep` + `fd` as dependencies, so
  `brew install vulcanshen/tap/filu` pulls in Search's tools; `install.sh` prints
  install hints for them when they're missing.
- Search (`/`, panel [2]): a native file finder (snacks/Telescope form, not the
  fzf binary) — a split popup with the file list on the left and a preview of the
  selected file on the right (stacked when narrow). It lists everything under the
  active tab's directory; typing filters that list by CONTENT via `ripgrep`
  (`--files-with-matches`, so a file that matches many times still appears once).
  Enter hands focus to the list; a second Enter reveals the pick in panel [2]. The
  preview scrolls to the matched line and marks it with a lavender bar. `fd` (or a
  Go walk) provides the initial file list.
- Panel [2] now shows a green tick on files sitting in the carries bucket, so a
  Pick (`p`) reads the same as it does in the Carries tab — and picking several
  files gives a visible multi-select before you Copy/Move them.
- Yank from panel [3]: `y` opens a viewport over the Preview/Meta content with a
  vim-style cursor and character-wise visual selection (`v`); `y` copies the
  selection, or the whole content when nothing is selected (OSC 52 + toast).
- cd-on-quit: `q` now opens a picker to leave the shell in panel [1]'s launch
  directory or any of the three list tabs' current directories (1–4 or j/k +
  Enter; Esc stays). With the shell wrapper installed (`eval "$(filu shell)"`),
  the shell cd's there on exit; `Ctrl+C` still hard-quits without cd-ing.
- Edit (`e`) opens a text file in `$EDITOR` inside an embedded PTY popup — the
  editor renders within filu instead of taking over the screen. Non-text files
  fall back to OS open; the directory reloads when the editor exits.
- Yank (`y`) in panel [2] or the Carries tab copies the item's full path to the
  system clipboard via OSC 52 (works over SSH and tmux), with a toast
  confirmation.
- Sort panel [2] by name / size / modified / extension with `S` (or the Space
  menu). A kbu-style column → direction picker builds a multi-tier sort chain
  (later tiers break ties), with per-column unset and a reset; directories stay
  first, the active sort shows in the Files header, and it persists per session.
- Pressing Enter on a file opens it in the OS default app (macOS `open`, Linux
  `xdg-open`); Enter on a directory still descends into it.
- Live refresh: the list tabs now watch their directories (fsnotify) and reload
  automatically when files are added or removed externally, keeping the cursor
  on its entry. Bursts are debounced into a single reload.

### Changed
- Panel [2] tabs are now dynamic: it opens with a single directory tab, `t`
  opens a new tab in the current directory (up to three) and `w` closes the
  active one — replacing the old fixed three tabs. Tabs are labelled with Roman
  numerals (Ⅰ / Ⅱ / Ⅲ) rather than the directory basename, since the active
  tab's full path already shows in the header bar; Zoom lays out however many
  tabs are open. The extra tabs you open persist across sessions, but the first
  tab always reopens at the launch directory and is active on start.
- The cd-on-quit picker now lists distinct directories — the launch dir plus each
  tab's dir with duplicates dropped — instead of one row per tab, so a directory
  open in several tabs is offered once.
- File colours now match your terminal's `eza` / `ls` exactly. filu bakes in a
  `vivid generate catppuccin-mocha` `LS_COLORS` palette (a colour per extension)
  and resolves it in eza's order — directory, symlink, executable (which beats an
  extension, so a `.sh` script reads as executable, not source), longest
  filename-suffix (e.g. `go.mod`), then the plain extension. No `LS_COLORS` is
  needed at run time, so every install shows the same palette.
- Panel [1]: the `Pinned` section title is now lavender and pinned entries render
  in the same colour as the system places (no longer lavender); a pinned
  directory shows a compact path (`~/Documents/sideproj/filu` → `~/D/s/filu`,
  `…`-trimmed from the left when it overflows); the whole panel dims when it is
  not focused. The pinned icon and the local-CWD icon were also updated.
- Panel [2] rows are back to `<space><icon><space><name>`; the carries tick now
  sits in that leading cell instead of adding an extra column.
- Text-entry fields (search, rename, add) now share one style: a peach chevron
  prompt and a blinking block cursor, no background bar. The rename popup shows
  the item exactly as panel [2] does — its type icon in its eza colour — with a
  grey divider under the input.
- The Space menu wraps a long action hint onto a continuation line instead of
  overflowing the box, and popup titles/hints that exceed the box are clipped.
- Line-number gutters now show just the number and a space — the "│" separator
  is gone — in both the text/binary preview and the panel [3] yank viewport, so
  the two share one form. The yank popup's content also starts flush against the
  top border (no leading blank row), matching kbu's YAML popup. The panel [3]
  Space menu's Yank entry no longer lists `v`, since visual is a within-popup
  action (already shown in the popup's hint bar).
- Rename moved to lowercase `r` (Delete stays `D`, to avoid clashing with the
  `d` half-page-down motion).
- On launch, panel [2] always opens on the first tab at the current directory,
  and panel [1] always lands on the current directory. The first tab's state,
  the active-tab index, and the places cursor are no longer persisted (tabs [2]
  and [3], focus, detail tab, carry, pinned, tasks, and sort still are).
- The Carries tab shows each item's full path (home folded to `~`), trimmed from
  the left so the filename stays visible, instead of just the basename.
- Panel [2] keys: Carry is now Pick (`p`) and Paste-here is now Copy (`c`), so
  `p` means "pick into the carries bucket" in both panel [2] and the Carries
  tab; the Carries tab's Pick also moved from `P` to `p`, off panel [1]/[2]'s
  `P` (Pin/UnPin).
- Pressing `q` while a copy/move is still running now asks for confirmation
  before quitting; `Ctrl+C` still force-quits immediately.

### Fixed
- Filenames containing control characters no longer shatter the layout. A macOS
  custom-icon file is literally named "Icon\r" (a carriage return), and drawing
  that CR reset the terminal cursor mid-line, misaligning every panel. Names are
  now stripped of control characters for display (the real name is kept for file
  operations) everywhere they're shown — list, tree/archive preview, places,
  carry, meta, header path, tab labels, and the rename input.
- Rename popup: the item name is now a description line inside the box (the
  border title stays "Rename"), and the full-width dark input bar starts the
  cursor flush to the pre-filled name — no untouchable leading blank.
- Panel borders no longer break on CJK Nerd Fonts (e.g. Maple Mono NF CN) that
  draw file-type icons two cells wide. filu now probes the terminal at startup
  (CPR) to measure the real icon cell width and reserves layout space to match.

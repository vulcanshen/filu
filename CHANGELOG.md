# Changelog

## [Unreleased]

## [0.1.0] — 2026-07-27

### Added
- A top status bar under the header shows the active tab's directory status:
  eza-coloured permissions (read yellow, write red, execute green), owner:group,
  item and hidden counts, and free / total disk on the right. The stat and
  statfs are computed once when the directory loads (never per frame — no
  recursive size sum), and the counts read live from the loaded list.
- Breadcrumb (`b` in panel [2], or the Space menu's `[b]readcrumb`): a popup
  listing the active tab's ancestor directories (root at top, current at the
  bottom); `j`/`k` move, Enter jumps the tab up to that ancestor, Esc/`b`/Space
  close. The cursor starts on the current level, so Enter is a no-op until you
  move.
- Goto (`go` in panel [2], or the Space menu's `[go]to`): a finder over `$HOME`
  that lists directories only and fuzzy-matches the path, so Enter teleports the
  active tab anywhere under home. The selected directory previews as its file
  tree. Reached with the `go` chord.
- A user config file, `config.yaml`, next to `state.yaml` under the OS config dir
  (macOS: `~/Library/Application Support/filu/`), written with a commented
  template on first run (an existing file is never overwritten). `finder_cap`
  (default 50000) bounds how many entries a finder scans — Goto walks all of
  `$HOME`, so raise it for reach or lower it if the fuzzy filter lags on a big
  home. `ignore_dirs` lists tool-generated directories the finders skip
  (`node_modules`, the Go module cache `go/pkg`, `OrbStack`, `Library`, `.git`,
  `.idea`, …); an entry matches that name anywhere, or a path when it has a slash.
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
  directory or any list tab's current directory (pick by number, or j/k +
  Enter; Esc stays). With the shell wrapper installed (`eval "$(filu shell)"`),
  the shell cd's there on exit; `Ctrl+C` still hard-quits without cd-ing.
- Yank (`y`) in panel [2] or the Carries tab copies the item's full path to the
  system clipboard via OSC 52 (works over SSH and tmux), with a toast
  confirmation.
- Open (`o`) opens the cursor file or directory with the OS default app.
  Open-with (`O`) opens a picker instead: **Default** plus the apps you configure
  under `open_with` in `config.yaml` (each a name + command; filu runs
  `<cmd> <path>`), so opening the current folder in your IDE is a keystroke away.
  This is how things are opened in filu — Enter is navigation only.
- Drop to a shell: `s` opens `$SHELL` (else `/bin/sh`) in the active tab's
  directory inside the embedded terminal — a modal sub-shell you leave by
  `exit`, after which the directory reloads. Quick per-tab work no longer needs
  quitting filu.
- Enter descends into a directory. In filu, Enter is navigation only — opening a
  file is not Enter's job (that role is the `[o]pen` / open-with menu); a file row
  Enter is a no-op.
- New tab at a chosen directory: `T` (shift-t) opens the Goto finder and, on
  Enter, opens the picked directory as a new panel [2] tab (`t` still opens one in
  the current directory). Both toast when the tab limit (5) is reached instead of
  silently doing nothing.
- Live refresh: the list tabs now watch their directories (fsnotify) and reload
  automatically when files are added or removed externally, keeping the cursor
  on its entry. Bursts are debounced into a single reload.
- A `Makefile` for local development: `build` (static `./filu`), `install` /
  `uninstall` (into `$GOPATH/bin`), `run`, `package` (a `.tar.gz` for the current
  platform under `dist/`), `test` / `vet` / `fmt` / `fmt-check` / `check`, and
  `clean`. Cross-platform release binaries still come from goreleaser; the
  Makefile only builds for the local machine.
- A product icon (`docs/icon.svg`) — a pixel-art mark spelling filu — now shows at
  the top of both READMEs, with a matching GitHub social-preview card
  (`docs/social-preview.png`).
- Demo GIFs in both READMEs: getting around, carry-bucket across tabs, the
  streaming finders, preview-then-yank, and a per-tab shell.
- `FILU_CONFIG` / `FILU_STATE` environment variables override where the config
  and state files live, for isolated runs (and the demo recordings).

### Changed
- The Carries tab's pick mark (an item in the land subset) now uses its own
  check-circle glyph, distinct from panel [2]'s carry-bucket mark, so the two
  "picked" states never read as the same thing.
- Panel [3]'s Preview and Meta tabs are split into two separate panels: `[3]`
  Preview keeps the right column's top 2/3, and a new `[5]` Meta panel takes the
  bottom 1/3 (mirroring the `[1]`/`[2]` over `[4]` split on the left). There is
  no more Preview/Meta tab toggle — both are always visible, each with its own
  scroll, yank, and zoom. Tab-cycling and the number keys now span five panels
  (`1`–`5`), and the session no longer persists a detail-tab choice.
- Panel [1]'s pinned directories now shorten with the same progressive scheme as
  the header breadcrumb (full → front-segment initials → middle `…` keeping root
  + folder name), instead of the old always-initials + left-truncate. The path
  fitting is one shared helper now.
- Panel [1]'s startup-directory entry is now labelled `LaunchDir` (was `CWD`),
  and the cd-on-quit picker names that destination with the same glyph + label
  (`<CWD glyph> LaunchDir`) instead of the terser `panel 1 (launch)`.
- The panel [2] pick mark is now a filled check-circle glyph (was a plain tick),
  and every row reserves the mark cell plus a following space — so picking a file
  only swaps blank↔glyph in place: the icon no longer shifts or butts against the
  mark.
- The header path bar is now a powerline breadcrumb instead of a flat path: a
  tab-numeral + folder-glyph chip, then one chip per path segment coloured along
  a continuous crust→blue gradient (interpolated by depth: root = crust, current
  directory = blue), the bar dark from the first cell. The wide contrast keeps
  the triangle separators visible between adjacent segments, and each segment's
  text flips dark/light by WCAG contrast so names stay legible across the span.
  When it overflows, front segments shrink to their initial (`~/Documents/x` →
  `~/D/x`), then the middle collapses to `…` keeping root + tail.
- The finder (Search / Find / Goto) now streams its listing: results appear as
  `fd` emits them, in fd's traversal order, instead of waiting for the whole walk
  and then showing a sorted list. The first results show almost immediately and
  filtering works while it loads. Goto is no longer ordered by modification time —
  a filesystem mtime turned out to track "an OS/tool touched this" (a removed
  Finder-sidebar entry, a dropped `.DS_Store`) rather than "you want to jump
  here", so it was a poor signal; you type to find the directory instead.
- The finder's list and preview boxes now hug their borders (no leading blank
  row above the input or the preview), reclaiming those rows for content.
- Jump-to-top is now the vim `gg` chord (was a single `g`) in panels [2]/[3]/[4]:
  a lone `g` arms and waits for the second key, matching kbu and freeing the `g`
  prefix for `go` (Goto). `G` still jumps to the bottom; the Space menu and the
  finder result list keep a single `g`.
- Every finder now shows a preview. Search (`/`) gained the split list+preview
  that Find (`f`) already had — it previews the selected file from the top — so
  Search and Find now differ only in what they match (name vs content), not in
  layout. (Goto previews the selected directory's tree.)
- In a finder's result list, `Esc` now leaves the finder — matching every other
  popup in the app — and `q` returns to the input to refine the query. Previously
  `Esc` in the list dropped back to the input, which read inconsistently.
- Search (`/`, panel [2]) is now a by-name fuzzy finder and content search moved
  to a new Find (`f`). Both share the same
  picker over the current tab's subtree: `/` fuzzy-matches file names in memory
  and ranks the best matches first (word-boundary, contiguous, and basename hits
  score highest); `f` filters by content via `ripgrep` and keeps the split
  preview that scrolls to the matched line. Enter reveals the pick and descends
  the tab into its directory.
- Panel [2] tabs are now dynamic: it opens with a single directory tab, `t`
  opens a new tab in the current directory (up to five) and `w` closes the
  active one — replacing the old fixed three tabs. Tabs are labelled with Roman
  numerals (Ⅰ … Ⅴ) rather than the directory basename, since the active
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
- Zoomed panel [2] (`z`): a tab's Roman numeral is no longer clipped on its right
  edge by the chip's rounded cap. The zoom chips sat flush against the cap —
  unlike the normal tab bar, which pads each label — so a wide glyph (Ⅱ/Ⅲ/Ⅳ) lost
  a sliver; a padding cell now guards it.
- Hardened the embedded terminal's shutdown: its reader goroutine and `stop()` no
  longer race over the shared PTY handles when a shell or editor session exits.

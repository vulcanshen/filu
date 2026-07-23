# Changelog

## [Unreleased]

### Added
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

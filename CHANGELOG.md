# Changelog

## [Unreleased]

### Added
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
- The Carries tab shows each item's full path (home folded to `~`), trimmed from
  the left so the filename stays visible, instead of just the basename.
- The Carries tab's Pick action is now lowercase `p` (was `P`), decoupling it
  from panel [1]/[2]'s `P` (Pin/UnPin).
- Pressing `q` while a copy/move is still running now asks for confirmation
  before quitting; `Ctrl+C` still force-quits immediately.

### Fixed
- Panel borders no longer break on CJK Nerd Fonts (e.g. Maple Mono NF CN) that
  draw file-type icons two cells wide. filu now probes the terminal at startup
  (CPR) to measure the real icon cell width and reserves layout space to match.

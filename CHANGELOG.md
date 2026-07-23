# Changelog

## [Unreleased]

### Added
- Pressing Enter on a file opens it in the OS default app (macOS `open`, Linux
  `xdg-open`); Enter on a directory still descends into it.

### Changed
- The Carries tab shows each item's full path (home folded to `~`), trimmed from
  the left so the filename stays visible, instead of just the basename.

### Fixed
- Panel borders no longer break on CJK Nerd Fonts (e.g. Maple Mono NF CN) that
  draw file-type icons two cells wide. filu now probes the terminal at startup
  (CPR) to measure the real icon cell width and reserves layout space to match.

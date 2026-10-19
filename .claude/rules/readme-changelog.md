---
description: README and CHANGELOG must be updated before every git commit and git tag
globs: *
---

# README & CHANGELOG Pre-Commit Rule

Before executing `git commit` or applying a `git tag`, ALWAYS perform the following checks and updates. Do not skip this step.

## README — what the user needs, as tdp E7 lays it out

`README.md` (English) and `README-zh_TW.md` (Traditional Chinese) say the same thing. Before a commit that changes what the user
sees or does, re-read the project (`cmd/filu`, `internal/ui`, the config schema, the keys each panel registers) and update both so
they match it: no stale features, keys, commands or settings.

The 12 sections of tdp E7, all of them, in this order:

1. Title, badges, language switch (`English · 繁體中文`)
2. One line on what filu is, then a paragraph on what it does
3. **One** representative demo gif
4. Features: what the user gets, not how it is done
5. Install: requirements, ways to install, what the first start does, uninstall
6. Getting started
7. Usage and keys
8. Settings and where things are kept
9. Limits, in the user's words
10. Links: CHANGELOG, `docs/dev-remarks.md`
11. terminu family: follows terminu design, lists the other members
12. License

- What belongs to a section goes in it as a subsection (updating next to uninstalling, in Install).
- A section of filu's own goes between 4 (Features) and 9 (Limits), after the one it is closest to; troubleshooting goes right
  after Usage. 1–3 and 10–12 are fixed.
- Section 3 has one gif; other sections get a gif or a screenshot only when they really need one.
- No "status" section with a version in it: the badge and the CHANGELOG carry the version.
- The settings section names the family's shared keys filu uses (`input_tab_width`, `hide_secrets_in_preview`) and their defaults;
  the Nerd Font requirement names `FILU__ICON_WIDTH`, `TERMINU__ICON_WIDTH`, `FILU__POWERLINE_WIDTH`, `TERMINU__POWERLINE_WIDTH`.
- Keys are written as tdp M5 writes them (`` `Enter` ``, `` `j|k` ``); labels from the screen as the screen shows them (`[r]ename`).
- How it works, why, building, releasing: `docs/dev-remarks.md`, not the README. The splash is never mentioned (tdp S2).

## CHANGELOG.md — Diff-Based Update

**Approach: diff comparison against previous commit or tag.**

**Scope: the CHANGELOG records changes to the filu program itself (the binary) only.**
Install/packaging tooling that ships independently of the versioned binary —
`install.sh`, `uninstall.sh`, the goreleaser/Homebrew/CI plumbing, and repo docs
(`README*`, `docs/`, `.claude/`) — does NOT get a CHANGELOG entry, even when the
change is user-facing. Those are delivered off `main` directly, not through a
tagged release, so they don't belong in a version's changelog. If a commit only
touches such tooling/docs, skip the CHANGELOG step entirely.

The format is tdp project's (the release notes are this file's section for the tag, cut out by `release.yml`):

### For `git commit`:
1. Run `git diff --staged` to see what changed
2. If the change is program-body (binary) behavior, add a concise entry to the
   `## Unreleased` section (open one at the top when there is none); if it only
   touches install tooling / packaging / docs (see Scope above), skip this step
3. Write in English, for the user: what changes for them, not how it is done.
   Group entries by type when there are several: `Added`, `Changed`, `Fixed`, `Removed`
4. Never mention the splash (tdp S2)
5. Skip this step on the very first commit (nothing to diff against)

### For `git tag vX.Y.Z`:
1. Rename `## Unreleased` to `## vX.Y.Z — tdp vA.B.C`: the tag, then the tdp version filu follows
2. Do not open a new `## Unreleased` until the next change comes in
3. Headings of releases before tdp v0.7.0 (`## [0.4.0] — 2026-10-06` and older) stay as they are

## Format Reference

```markdown
# Changelog

Each release names the version of terminu design (tdp) it follows.

## Unreleased

### Added
- …

## v0.5.0 — tdp v0.8.0

### Changed
- …

## [0.4.0] — 2026-10-06
- …
```

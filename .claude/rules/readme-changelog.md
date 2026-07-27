---
description: README and CHANGELOG must be updated before every git commit and git tag
globs: *
---

# README & CHANGELOG Pre-Commit Rule

Before executing `git commit` or applying a `git tag`, ALWAYS perform the following checks and updates. Do not skip this step.

## README.md — Full Project Scan & Rewrite

**Approach: complete project scan, not diff comparison.**

### On initial scaffold commit (no source code yet):
Use CLAUDE.md and the project's stated description/objectives as the basis. Write a first draft that accurately describes intent, planned features, and build/test commands. Do not leave placeholder sections blank — fill them with what is known.

### On all subsequent commits:
1. Read all source files (main entry points, core modules, config files, public API)
2. Read `CLAUDE.md` for project intent and architecture
3. Run `git log --oneline --since="7 days ago"` for recent activity context
4. Rewrite `README.md` from scratch based on current project state

README.md must accurately reflect:
- What the project does (no stale feature lists)
- Installation / build instructions (match actual current commands)
- Usage examples (match current CLI/API surface)
- Configuration (match current config schema)
- Any section that existed before: re-evaluate and rewrite, do not blindly keep old content

**Rewriting is preferred over incremental edits** — if the project changed significantly, a full rewrite produces a more coherent result than accumulated patches.

## CHANGELOG.md — Diff-Based Update

**Approach: diff comparison against previous commit or tag.**

**Scope: the CHANGELOG records changes to the filu program itself (the binary) only.**
Install/packaging tooling that ships independently of the versioned binary —
`install.sh`, `uninstall.sh`, the goreleaser/Homebrew/CI plumbing, and repo docs
(`README*`, `docs/`, `.claude/`) — does NOT get a CHANGELOG entry, even when the
change is user-facing. Those are delivered off `main` directly, not through a
tagged release, so they don't belong in a version's changelog. If a commit only
touches such tooling/docs, skip the CHANGELOG step entirely.

### For `git commit`:
1. Run `git diff --staged` to see what changed
2. If the change is program-body (binary) behavior, add a concise entry to the
   `## [Unreleased]` section; if it only touches install tooling / packaging /
   docs (see Scope above), skip this step
3. Group entries by type: `Added`, `Changed`, `Fixed`, `Removed`
4. Keep entries user-facing — describe behavior change, not implementation detail
5. Skip this step on the very first commit (nothing to diff against)

### For `git tag vX.Y.Z`:
1. Move all entries from `## [Unreleased]` to a new `## [X.Y.Z] — YYYY-MM-DD` section
2. Keep `## [Unreleased]` header (empty) at the top for future changes
3. Do NOT add a new Unreleased entry — leave it clean for the next cycle

## Format Reference

```markdown
# Changelog

## [Unreleased]
### Added
- ...

## [1.2.0] — 2026-05-16
### Added
- New feature X
### Fixed
- Bug in Y
```

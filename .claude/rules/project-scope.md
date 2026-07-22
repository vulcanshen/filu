---
description: Enforced rules for filu
globs: *
---

# filu Enforced Rules

## 1. Project-Scoped Configuration Only
All Claude settings, configurations, and customizations MUST stay within this project's `.claude/` directory.
NEVER modify global settings in `~/.claude/`.
If a setting needs to be changed, use the project-scoped file at `.claude/settings.json`.

## 2. Write Boundary
- `Edit` and `Write` tools: stay within this project directory (the filu repo). Never write to an external absolute path.
- Bash write/destructive commands (rm, cp, mv, `echo >`, curl POST, git push, gh, etc.) are NOT auto-allowed — they prompt for confirmation each time.
- NEVER execute runtime interpreters or run the app (`go run`, node, python3, …) without user approval.

## 3. Deny List (Absolute Block)
NEVER permitted, even if asked — warn and refuse:
- `rm -rf /` or `rm -rf ~` (system/home destruction)
- `git push --force` / `git push -f` (remote history destruction)
- `git reset --hard` (local history destruction)
- `git clean -f` (untracked file destruction)
- `chmod 777` (security degradation)
- Writing secrets (API keys, tokens, passwords) to any tracked file

## 4. Session Checkpoint Reminder
Before ending a session, compacting, or clearing context, remind the user to run `/save-checkpoint` to preserve session history.

## 5. Language
Respond in Traditional Chinese (繁體中文) unless the context requires English (code, commits, config files).

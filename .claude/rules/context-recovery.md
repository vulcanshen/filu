---
description: Context loss recovery protocol for goal-driven projects
globs: *
---

# Context Recovery Protocol

When context is lost or compacted, work must survive and resume seamlessly using checkpoint files and progress tracking.

## Path Convention

All forge meta files live under **`.forge/meta/`**. References to `GOAL.md` / `PROGRESS.md` / `IDEA.md` in this document are shorthand for `.forge/meta/GOAL.md` etc. Checkpoints live in `.checkpoints/`.

## What Triggers Recovery

- Session ended unexpectedly
- Context window compacted (`/compact` command)
- Browser reload or session restart
- Claude Code reconnection after network loss

## Automatic Recovery with `/load-checkpoints`

The `/load-checkpoints` skill automates the recovery process:

1. Scans `.checkpoints/` directory for all checkpoint files
2. Identifies the most recent checkpoint (by timestamp)
3. Reads and displays: Summary, Key Decisions, Current State, Next Steps
4. Suggests exact resume point based on checkpoint content

**Usage:**
```bash
/load-checkpoints
```

## Manual Recovery (if `/load-checkpoints` unavailable)

### Step 1: Read PROGRESS.md
```
PROGRESS.md is the definitive state record:
- Current Phase — what stage of development
- Last Completed — what was just finished
- In Progress — what was being worked on (if interrupted mid-task)
- Next Up — what should be done next
- Blockers — any questions needing user decision
```

### Step 2: Read GOAL.md
```
GOAL.md contains all objectives and acceptance criteria.
Compare checkboxes against PROGRESS.md to find exact stopping point.
```

### Step 3: Review Recent Commits
```bash
git log --oneline -10
```
Latest commits show exactly what was completed since last checkpoint.

### Step 4: Resume
- If a feature was partially completed → finish it
- If tests were failing → continue fixing
- If blocked on user input → ask again
- Otherwise → move to next unchecked task

## Checkpoint File Format

Checkpoints are stored at `.checkpoints/<timestamp>-<session-id>.md` with mandatory sections:

```markdown
# Session Checkpoint — 2026-05-18_14:32:45

## Summary
One-paragraph summary of what was accomplished.

## Key Decisions
- Decision 1 and rationale
- Decision 2 and rationale

## Changes Made
- File 1: what changed
- File 2: what changed

## Current State
- What is done
- What is in progress
- What's next

## Next Steps
- Immediate next action
- Open questions
- Blockers

## Context Notes
- Architecture decisions that matter
- Dependencies that need attention
- Any gotchas discovered
```

## Quality Gates for Resumption

Before resuming work, verify:
- [ ] Latest checkpoint timestamp is recent
- [ ] PROGRESS.md matches last checkpoint state
- [ ] `git log` confirms all committed work
- [ ] No uncommitted changes (unless in-progress task)
- [ ] GOAL.md checkboxes are clear and unambiguous

## Preventing Unnecessary Recovery

To avoid context loss:
- Run `/save-checkpoint` before long breaks
- Run `/save-checkpoint` before compacting context (`/compact`)
- Commit frequently — every completed feature gets a commit
- Update PROGRESS.md after every significant change

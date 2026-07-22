---
description: Goal-driven autonomous execution rules
globs: *
---

# Goal-Driven Execution Rules

## Path Convention

All forge meta files live under **`.forge/meta/`** at the project root:

- `.forge/meta/GOAL.md`
- `.forge/meta/PROGRESS.md`
- `.forge/meta/IDEA.md` (if scaffolded from an IDEA.md)

Throughout this document `GOAL.md` and `PROGRESS.md` are shorthand for the full paths above. The shorthand exists for readability — when reading or writing, always use the `.forge/meta/` path.

## 1. GOAL.md is the Source of Truth
All work is driven by `GOAL.md` (i.e. `.forge/meta/GOAL.md`). Before starting any task, read it to understand current objectives and acceptance criteria.

## 2. Roles

### Orchestrator (Main Loop)
- Reads GOAL.md, determines current state and next task
- Executes sequential tasks directly: implement → test → fix → next
- Spawns Feature Agents for parallelizable, non-overlapping work
- Runs integration tests after parallel work completes
- Updates GOAL.md checkboxes and PROGRESS.md after each iteration
- Commits progress after each completed feature

### Feature Agent
- Receives ONE clearly scoped task with explicit file ownership
- Self-contained: develop → test → debug → test pass → report
- MUST NOT modify files outside its assigned scope
- If shared code has a bug → do NOT fix it, report to Orchestrator
- Reports completion via committed code, not just context

## 3. Progress Persistence (Survive Context Loss)

Context can be compacted or lost at any time. Progress MUST live in files, not in context.

### Every Loop Iteration Must:
1. Update `GOAL.md` checkboxes — the definitive progress record
2. Update `PROGRESS.md` — current state, what was just done, what's next
3. `git commit` all changes — every feature completion is a commit

### Resuming After Context Loss:
1. Run `/load-checkpoints` for automatic recovery
2. Or manually: read `GOAL.md`, `PROGRESS.md`, latest `.checkpoints/*.md`, then `git log`

### PROGRESS.md Format:
```markdown
# Progress

## Current Phase
Description of current phase

## Last Completed
- What was just finished

## In Progress
- What was being worked on (if interrupted)

## Next Up
- What should be done next

## Blockers
- Any issues that need user decision
```

## 4. Autonomous Loop Behavior
When a `/loop` is active:
1. Read `GOAL.md` and `PROGRESS.md` to determine current state
2. Pick the next unchecked task
3. If tasks are parallelizable: spawn Feature Agents with non-overlapping file ownership
4. Otherwise: implement → test → fix → test pass → mark done
5. Update `GOAL.md` checkbox + `PROGRESS.md` + git commit
6. Move to next task
7. Repeat

## 5. Stopping Conditions
Stop the loop and notify the user when ANY of the following is true:
- All acceptance criteria in `GOAL.md` are checked
- A blocking question arises that requires user decision
- A task would violate security rules in `project-scope.md`
- Context feels large — proactively commit and update PROGRESS.md before it's too late

## 6. Quality Gates
Before checking off any acceptance criterion:
- The implementation must actually work (not just exist)
- If the criterion is testable, tests must pass
- Code must not break previously completed criteria (no regressions)
- Build must succeed after every change

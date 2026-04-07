---
id: add-worktreecreate-worktreeremove-9373
stage: triage
deps: []
links: []
created: 2026-04-07T03:18:44Z
type: feature
priority: 3
assignee: Dustin Reynolds
version: 1
---
# Add WorktreeCreate and WorktreeRemove hook events


Fire WorktreeCreate when a worktree is being created (via --worktree or isolation: worktree) and WorktreeRemove when one is being removed. Enables custom git worktree management, replacing or augmenting default behavior.

## Scope
- Add WorktreeCreate and WorktreeRemove to HookEventName constants in internal/hooks/types.go
- Wire into the worktree lifecycle in the app/agent layer
- Payload: session_id, RawEventData with worktree path and branch

## Acceptance Criteria
- WorktreeCreate fires before worktree creation
- WorktreeRemove fires before worktree removal
- Both are non-blocking (async)
- Payload includes session_id, data.path, data.branch

---
id: add-cwdchanged-hook-0af6
stage: done
deps: []
links: []
created: 2026-04-07T03:18:44Z
type: feature
priority: 3
assignee: Dustin Reynolds
skipped: [spec, design, implement, test, verify]
version: 2
---
# Add CwdChanged hook event

Fire CwdChanged when the working directory changes (e.g. when the agent executes a cd command via Bash). Enables reactive environment management with tools like direnv, environment reloading, or path-sensitive hook logic.

## Scope
- Add CwdChanged to HookEventName constants in internal/hooks/types.go
- Identify where the working directory is tracked/changed in the agent tools layer (Bash tool)
- Fire async with old and new working directory in payload
- No matcher support (always fires on every change)

## Acceptance Criteria
- Fires whenever the working directory changes during a session
- Non-blocking (async)
- Payload includes session_id, data.cwd (new dir), data.previous_cwd (old dir)

---
id: add-sessionend-hook-0876
stage: done
deps: []
links: []
created: 2026-04-07T03:18:44Z
type: feature
priority: 2
assignee: Dustin Reynolds
skipped: [design, implement, test, verify]
version: 3
---
# Add SessionEnd hook event

Fire a SessionEnd hook when a session terminates. Natural bookend to SessionStart. Enables cleanup scripts, audit logging, and session duration tracking.

## Scope
- Add SessionEnd to HookEventName constants in internal/hooks/types.go
- Identify the session termination point in the agent/app layer
- Fire non-blocking (async) with session_id in payload

## Acceptance Criteria
- Fires once when a session ends (normal exit, cancel, or error)
- Non-blocking
- Payload includes session_id

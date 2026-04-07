---
id: add-teammateidle-hook-bfde
stage: triage
deps: []
links: []
created: 2026-04-07T03:18:44Z
type: feature
priority: 4
assignee: Dustin Reynolds
version: 1
---
# Add TeammateIdle hook event


Fire TeammateIdle when an agent team teammate is about to go idle. Relevant for multi-agent coordination scenarios.

## Scope
- Add TeammateIdle to HookEventName constants in internal/hooks/types.go
- Wire into agent team idle detection logic (if present)
- Payload: session_id, RawEventData with teammate identifier

## Acceptance Criteria
- Fires when a teammate agent is about to go idle
- Non-blocking (async)
- Payload includes session_id and data.teammate_id

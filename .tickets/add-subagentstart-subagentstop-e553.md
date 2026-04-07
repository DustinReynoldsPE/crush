---
id: add-subagentstart-subagentstop-e553
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
# Add SubagentStart and SubagentStop hook events

Fire SubagentStart when a subagent is spawned and SubagentStop when it finishes. Enables monitoring, logging, and resource tracking for multi-agent workflows.

## Scope
- Add SubagentStart and SubagentStop to HookEventName constants in internal/hooks/types.go
- Identify where subagents are spawned in the codebase (isSubAgent flag, coordinator)
- Fire async hooks at spawn and completion points
- Payload: session_id, RawEventData with agent type/name

## Acceptance Criteria
- SubagentStart fires when a subagent session begins
- SubagentStop fires when a subagent session ends
- Both are non-blocking (async)
- Payload includes session_id and data.agent_type

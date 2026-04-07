---
id: add-stopfailure-hook-bb67
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
# Add StopFailure hook event

Fire a StopFailure hook when the agent turn ends due to an API/provider error (rate limit, auth failure, billing, server error, etc.). This is the error-path bookend to the Stop hook and distinct from AgentError — it fires specifically when the turn cannot complete cleanly rather than when stream processing fails mid-flight.

## Scope
- Add StopFailure to HookEventName constants in internal/hooks/types.go
- Fire non-blocking (async) in the error handling block of agent.go, after the finish reason is set to FinishReasonError, complementing the existing AgentError hook
- Payload: session_id, hook_event_name, RawEventData with error string and finish_reason

## Acceptance Criteria
- Fires on API errors, rate limits, billing errors, provider errors
- Does not fire on context.Canceled or permission denials
- Non-blocking; does not affect error propagation
- Payload includes session_id and data.error string

---
id: hook-context-window-ad68
stage: done
deps: []
links: []
created: 2026-04-06T12:00:00Z
type: feature
priority: 3
assignee: Dustin Reynolds
skipped: [design, implement, test, verify]
version: 3
---
# Add ContextWindowFull hook event

Fire a `ContextWindowFull` hook when the agent's context window threshold is hit and auto-summarization is about to begin. This is a significant state transition with no current observability hook. Enables alerting, dumping context to disk before summarization, or logging for analysis.

## Scope

- Add `ContextWindowFull` to `HookEventName` constants in `internal/hooks/types.go`
- In `internal/agent/agent.go`, locate the context window threshold check (the `StopCondition` that triggers `Summarize()`)
- Fire the hook **non-blocking** (async) just before `Summarize()` is called
- Payload: `session_id`, `hook_event_name: "ContextWindowFull"`, and `RawEventData` with `"tokens_used"` and `"threshold"` values

## TDD Steps

1. Write a unit test in `internal/hooks/manager_test.go` for the `ContextWindowFull` event name
2. Write an integration test that configures a very low context threshold, runs the agent until summarization triggers, wires a `ContextWindowFull` hook script, and asserts it fires once with the expected payload fields
3. Verify the hook fires before `Summarize()` is called, not after
4. Implement: add constant, wire at the threshold check, fire async
5. All tests pass, commit

## Acceptance Criteria

- `ContextWindowFull` fires exactly once when the context threshold is crossed
- Fires before summarization begins, not after
- Payload includes `tokens_used` and `threshold` in `RawEventData`
- Non-blocking; does not delay summarization
- Does not fire on normal agent stops where context is not full

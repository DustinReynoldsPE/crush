---
id: hook-agent-error-a252
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
# Add AgentError hook event

Fire an `AgentError` hook when `agent.Stream()` returns an error that is not a tool-level failure (i.e., not covered by `PostToolUseFailure`). Covers API failures, network errors, provider errors, and context cancellations. Enables alerting, fallback logic, and error telemetry without polling logs.

## Scope

- Add `AgentError` to `HookEventName` constants in `internal/hooks/types.go`
- In `internal/agent/agent.go`, identify the error handling block after `agent.Stream()` returns a non-nil error (excluding `permission.ErrorPermissionDenied` and `context.Canceled`, which are intentional stops)
- Fire the hook **non-blocking** (async) with the error details in `RawEventData`
- Payload: `session_id`, `hook_event_name: "AgentError"`, `RawEventData` with `"error": err.Error()`

## TDD Steps

1. Write a unit test in `internal/hooks/manager_test.go` for the `AgentError` event name
2. Write an integration test that injects a failing provider into the agent, wires an `AgentError` hook script, and asserts the script receives a JSON payload with a non-empty `error` field
3. Ensure `context.Canceled` and `permission.ErrorPermissionDenied` do NOT trigger `AgentError`
4. Implement: add constant, wire at the error handling site, fire async
5. All tests pass, commit

## Acceptance Criteria

- `AgentError` fires on genuine stream errors (API/network/provider failures)
- Does not fire on permission denials or intentional cancellations
- Payload includes `session_id` and `error` string in `RawEventData`
- Non-blocking; does not affect error propagation to the caller

<!-- checkpoint: testing -->
<!-- exit-state: AgentError constant added, hook wired async in agent.go error block (guarded by !cancel && !permissionDenied), 4 manager tests + 3 agent integration tests passing; ready for commit -->
<!-- key-files: internal/hooks/types.go, internal/agent/agent.go, internal/hooks/manager_test.go, internal/agent/agent_error_hook_test.go -->
<!-- open-questions: none -->

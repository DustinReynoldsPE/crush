---
id: hook-step-events-8f6e
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
# Add PreStep and PostStep hook events

Fire `PreStep` before each LLM API call and `PostStep` after each step completes (maps to `PrepareStep` and `OnStepFinish` callbacks in the agent). Enables per-step token/cost tracking, input logging, and observability tooling without patching the agent.

## Scope

- Add `PreStep` and `PostStep` to `HookEventName` constants in `internal/hooks/types.go`
- `PreStep`: fire in the `PrepareStep` callback, **non-blocking** (must not delay inference). Payload: `session_id`, `step_index` in `RawEventData`
- `PostStep`: fire in the `OnStepFinish` callback, **non-blocking**. Payload: `session_id`, `step_index`, `finish_reason`, `usage` (input/output tokens) in `RawEventData`
- Both fire on every step of a multi-step agent loop

## TDD Steps

1. Write unit tests in `internal/hooks/manager_test.go` for both `PreStep` and `PostStep` event names
2. Write an integration test that runs a two-step agent (one tool call + final response), wires both hooks, and asserts each fires the expected number of times with monotonically increasing `step_index`
3. Verify payload includes token usage fields on `PostStep`
4. Implement: add constants, wire at callback sites, fire async
5. All tests pass, commit

## Acceptance Criteria

- `PreStep` fires once per inference step, before the LLM call
- `PostStep` fires once per inference step, after `OnStepFinish`
- Both hooks are non-blocking (async)
- `PostStep` payload includes `finish_reason` and `usage` (tokens)
- A two-step agent loop produces exactly two `PreStep` and two `PostStep` events

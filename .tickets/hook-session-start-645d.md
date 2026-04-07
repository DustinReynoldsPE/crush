---
id: hook-session-start-645d
stage: done
deps: []
links: []
created: 2026-04-06T12:00:00Z
type: feature
priority: 2
assignee: Dustin Reynolds
skipped: []
version: 1
---
# Add SessionStart hook event

Fire a `SessionStart` hook once when a new session is created, before any user input is processed. Listed explicitly in hooks_goal.md as a target event. High utility: setup scripts, environment initialization, session metadata logging, injecting context.

## Scope

- Add `SessionStart` to `HookEventName` constants in `internal/hooks/types.go`
- Find the session initialization point in `internal/agent/agent.go` (session retrieved + messages fetched, before `UserPromptSubmit`)
- Fire the hook as **blocking** — a deny here should abort the session with the deny reason as the error message
- Pass `SessionID` and any available session metadata in `RawEventData`

## TDD Steps

1. Write a test in `internal/hooks/manager_test.go` that registers a `SessionStart` hook config, calls `manager.Execute(SessionStart, event)`, and asserts the hook script is invoked with the correct stdin JSON
2. Write an integration test in `internal/agent/agent_test.go` (or similar) that wires a deny-returning `SessionStart` hook and asserts `Run()` returns an error before any LLM call is made
3. Implement: add the constant, find the trigger point, fire the hook, handle deny
4. All tests pass, commit

## Acceptance Criteria

- `SessionStart` hook fires exactly once per `Run()` invocation, before `UserPromptSubmit`
- A deny result from the hook causes `Run()` to return an error with the hook's reason
- `HookEvent` payload includes `session_id` and `hook_event_name: "SessionStart"`
- Existing hook tests are unaffected

<!-- checkpoint: finalized -->
<!-- exit-state: Implementation complete and committed (8a3ed4f0). SessionStart hook fires in agent.go after getSessionMessages, before title-generation goroutine. 6 unit tests + 4 integration tests, all passing. -->
<!-- key-files: internal/hooks/types.go, internal/agent/agent.go:223-233, internal/hooks/manager_test.go, internal/agent/session_start_hook_test.go -->
<!-- open-questions: none -->

---
id: hook-notification-6dc6
stage: done
deps: []
links: []
created: 2026-04-06T12:00:00Z
type: feature
priority: 2
assignee: Dustin Reynolds
skipped: [design, implement, test, verify]
version: 3
---
# Add Notification hook event

Fire a `Notification` hook when the agent would surface a notification to the user (e.g., the "agent finished" event published in non-interactive mode). Listed in hooks_goal.md as a target event. Enables routing to Slack, ntfy, desktop notifiers, etc. without the user polling.

## Scope

- Add `Notification` to `HookEventName` constants in `internal/hooks/types.go`
- Identify where `PromptRespondedEvent` (or equivalent agent-finished notification) is published in `internal/agent/agent.go`
- Fire the hook **non-blocking** (async) at that point
- Payload should include `session_id`, `hook_event_name: "Notification"`, and a `message` field in `RawEventData` describing what triggered the notification (e.g., `"agent_finished"`)

## TDD Steps

1. Write a unit test in `internal/hooks/manager_test.go` verifying `Notification` is a recognized event name and that async execution is used (hook fires without blocking)
2. Write an integration test that wires a `Notification` hook script and asserts it receives the correct JSON after `Run()` completes in non-interactive mode
3. Implement: add constant, wire at publish point, fire async
4. All tests pass, commit

## Acceptance Criteria

- `Notification` hook fires after the agent publishes its "finished" event
- Hook is non-blocking (does not delay the response to the user)
- Payload includes `session_id` and a `message` field in `RawEventData`
- Existing hook and agent tests are unaffected

<!-- checkpoint: testing -->
<!-- exit-state: Notification hook constant added, async firing wired in agent.go after successful stream, 5 manager tests all passing; ready for commit -->
<!-- key-files: internal/hooks/types.go, internal/agent/agent.go, internal/hooks/manager_test.go -->
<!-- open-questions: none -->

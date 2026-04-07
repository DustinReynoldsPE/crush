---
id: hook-ui-feedback-afc0
stage: done
deps: [hook-session-start-645d]
links: []
created: 2026-04-06T13:00:00Z
type: feature
priority: 2
assignee: Dustin Reynolds
skipped: [design, implement, test, verify]
version: 3
---
# Hook execution visual feedback in TUI

Hooks currently fire silently. The user has no indication that a hook is
running, which means slow hooks make the UI appear frozen, and deny results
look like generic errors with no attribution. This ticket adds threshold-based
status bar feedback and explicit deny attribution for all hook types.

## Problem Statement

Three distinct failure modes today:

1. **Slow hook (> 300ms, proceeds)** — UI hangs with no explanation. User
   doesn't know if Crush is thinking, waiting on a model, or stuck on a hook.

2. **Hook deny** — `Run()` returns e.g. `"session start denied: blocked by
   policy"`. The phrase "session start denied" is parsed from an error string,
   not a structured event. The status bar shows it as a generic red error with
   no visual distinction from a crash or API failure.

3. **Fast hook (< 300ms, proceeds)** — invisible. This is **correct**. No
   UI change needed here.

## Design

### What NOT to do

- Do not add a chat-log entry for every hook execution. That pollutes the
  conversation history and trains users to ignore it.
- Do not block rendering or add a modal for hooks that proceed.
- Do not show anything for fast hooks (< 300ms). The threshold exists to
  suppress noise.

### What to do

**While running (slow hook):** Show a transient `InfoTypeInfo` status bar
message: `Running <HookType> hook…` Clear it when execution completes.

**On deny:** Show a `InfoTypeWarn` status bar message:
`<HookType> hook denied: <reason>` with a longer TTL (10s vs the default 5s)
so the user has time to read it. The existing error propagation path (which
causes the chat to show the deny reason inline) is preserved — this is
additive.

**On proceed (fast or slow):** Clear the running message silently. No
success flash.

**On error (hook subprocess failed):** Show `InfoTypeWarn`:
`<HookType> hook error: <reason>` — same TTL as deny.

### Architecture

The hook manager needs to publish lifecycle events that the app layer can
route to the TUI. This follows the exact same pattern as `notify.Notification`
and `agentNotifications`:

```
hooks.Manager
  └─ pubsub.Publisher[hooks.Notification]
       └─ app.hookNotifications (pubsub.Broker)
            └─ app.setupEvents() → app.events channel
                 └─ tea.Program.Send()
                      └─ ui.go Update() case pubsub.Event[hooks.Notification]
```

## Implementation

### Step 1 — `internal/hooks/notification.go` (new file)

Define the notification types that the manager will publish:

```go
package hooks

import "time"

// NotificationType identifies the lifecycle phase of a hook event.
type NotificationType string

const (
    // NotificationRunning is published when a blocking hook begins execution.
    // Only fired for synchronous (non-async) hooks.
    NotificationRunning NotificationType = "hook_running"
    // NotificationComplete is published when a hook finishes execution.
    NotificationComplete NotificationType = "hook_complete"
)

// Notification is a lifecycle event published by the hook Manager.
type Notification struct {
    Type      NotificationType
    HookType  HookType   // e.g. "SessionStart", "PreToolUse"
    Command   string     // the hook command (for logging/display)
    SessionID string
    // Complete-only fields:
    Decision string        // "proceed", "deny", "modify", "error"
    Reason   string        // populated on deny or error
    Elapsed  time.Duration // total wall time for this hook chain
}
```

### Step 2 — Inject publisher into `Manager`

Add an optional `pubsub.Publisher[Notification]` to the Manager. Use a
functional option so existing callers and tests are unaffected:

```go
type ManagerOption func(*Manager)

func WithPublisher(p pubsub.Publisher[Notification]) ManagerOption {
    return func(m *Manager) { m.publisher = p }
}

func NewManager(hooks map[HookType][]HookConfig, opts ...ManagerOption) *Manager {
    m := &Manager{executor: NewExecutor(), hooks: hooks}
    for _, o := range opts { o(m) }
    ...
}
```

Add `publisher pubsub.Publisher[Notification]` to the `Manager` struct. Add a
`publish(n Notification)` helper that is a no-op when publisher is nil (nil
check, no-op — keeps tests simple).

### Step 3 — Manager publishes events in `Execute()`

In `Manager.Execute()`, wrap synchronous hook execution:

```go
start := time.Now()
m.publish(Notification{
    Type:      NotificationRunning,
    HookType:  hookType,
    Command:   hookCfg.Command,
    SessionID: event.SessionID,
})

result, execErr := m.executor.Execute(ctx, hookCfg, event)

m.publish(Notification{
    Type:      NotificationComplete,
    HookType:  hookType,
    Command:   hookCfg.Command,
    SessionID: event.SessionID,
    Decision:  result.Decision,
    Reason:    result.Reason,
    Elapsed:   time.Since(start),
})
```

Only publish for synchronous hooks. Async hooks (fire-and-forget) must NOT
publish `NotificationRunning` because their completion is unobservable from the
main chain.

### Step 4 — Wire broker in `internal/app/app.go`

Add `hookNotifications pubsub.Broker[hooks.Notification]` to the `App` struct.
In `NewApp` (or wherever the broker is initialised), initialise it similarly to
`agentNotifications`. In `setupEvents()`:

```go
setupSubscriber(ctx, app.serviceEventsWG, "hook-notifications",
    app.hookNotifications.Subscribe, app.events)
```

In `InitCoderAgent` (or wherever `NewCoordinator` is called), pass the broker
down so the coordinator can inject it into the hooks manager via
`hooks.WithPublisher(app.hookNotifications)`.

Trace the exact call chain:
`app.hookNotifications` → `coordinator` → `hooks.NewManager(..., hooks.WithPublisher(...))`

### Step 5 — TUI handles `pubsub.Event[hooks.Notification]`

In `internal/ui/model/ui.go`, add a case to `Update()`:

```go
case pubsub.Event[hooks.Notification]:
    return m, m.handleHookNotification(msg.Payload)
```

Implement `handleHookNotification`:

```go
const hookSlowThreshold = 300 * time.Millisecond

func (m *UI) handleHookNotification(n hooks.Notification) tea.Cmd {
    switch n.Type {
    case hooks.NotificationRunning:
        // Only show if we cross the threshold — use a delayed cmd.
        return m.hookRunningCmd(n)
    case hooks.NotificationComplete:
        switch n.Decision {
        case "deny":
            return util.CmdHandler(util.InfoMsg{
                Type: util.InfoTypeWarn,
                Msg:  fmt.Sprintf("%s hook denied: %s", n.HookType, n.Reason),
                TTL:  10 * time.Second,
            })
        case "error":
            return util.CmdHandler(util.InfoMsg{
                Type: util.InfoTypeWarn,
                Msg:  fmt.Sprintf("%s hook error: %s", n.HookType, n.Reason),
                TTL:  10 * time.Second,
            })
        default:
            // Proceed or modify: clear any running message silently.
            return util.CmdHandler(util.ClearStatusMsg{})
        }
    }
    return nil
}
```

**Threshold implementation for `hookRunningCmd`:**

The naive approach (show immediately, hide on complete) causes a flash for
fast hooks. Instead, use a delayed command:

```go
// hookRunningMsg is sent after the threshold delay to show the running indicator.
type hookRunningMsg struct{ hookType hooks.HookType }

func (m *UI) hookRunningCmd(n hooks.Notification) tea.Cmd {
    return tea.Tick(hookSlowThreshold, func(time.Time) tea.Msg {
        return hookRunningMsg{hookType: n.HookType}
    })
}
```

In `Update()`, handle `hookRunningMsg`: only show the status bar message if
`m.status` does not already have a non-hook message visible (avoid clobbering
errors). The `NotificationComplete` event clears it via `ClearStatusMsg`.

**Race condition:** A complete event can arrive before the `tea.Tick` fires
(fast hook). The clear message fires first, then the tick fires and shows the
"running" indicator after the hook has already finished. Guard this with a
boolean `m.hookRunning bool` — set true on `NotificationRunning`, false on
`NotificationComplete`. In `hookRunningMsg` handler, only show if
`m.hookRunning` is still true.

```go
// In Update, on NotificationRunning:
m.hookRunning = true

// In Update, on NotificationComplete:
m.hookRunning = false

// In Update, on hookRunningMsg:
if m.hookRunning {
    m.status.SetInfoMsg(util.InfoMsg{
        Type: util.InfoTypeInfo,
        Msg:  fmt.Sprintf("Running %s hook…", msg.hookType),
        TTL:  0, // no auto-clear; cleared by NotificationComplete
    })
}
```

Set `TTL: 0` (no auto-clear) on the running message so it persists until the
complete event clears it. The existing `clearInfoMsgCmd` must not be fired for
`TTL == 0`. Add that guard in `Update()`:

```go
case util.InfoMsg:
    m.status.SetInfoMsg(msg)
    if msg.TTL > 0 {
        cmds = append(cmds, clearInfoMsgCmd(msg.TTL))
    }
```

### Step 6 — Tests

**`internal/hooks/manager_test.go`** — publisher integration:

```go
// TestManager_Publisher_RunningFiredBeforeComplete
// Use a fake Publisher[Notification] that records events in order.
// Assert: for a slow-ish hook (sleep 0.1), NotificationRunning arrives
// before NotificationComplete.

// TestManager_Publisher_AsyncHook_NoRunningEvent
// Async hooks must NOT publish NotificationRunning.

// TestManager_Publisher_NilPublisher_NoPanic
// NewManager with no WithPublisher option must not panic during Execute.

// TestManager_Publisher_DenyComplete_DecisionAndReason
// After a deny hook, NotificationComplete has Decision=="deny" and
// Reason matching the hook's stderr output.
```

Implement a `fakePublisher` in a new `publisher_test.go` or inline in
`manager_test.go`:

```go
type fakePublisher struct {
    mu     sync.Mutex
    events []hooks.Notification
}
func (f *fakePublisher) Publish(_ pubsub.EventType, n hooks.Notification) {
    f.mu.Lock(); defer f.mu.Unlock()
    f.events = append(f.events, n)
}
func (f *fakePublisher) Events() []hooks.Notification {
    f.mu.Lock(); defer f.mu.Unlock()
    return slices.Clone(f.events)
}
```

**`internal/ui/model/ui_test.go`** (or a new `hook_feedback_test.go`):

```go
// TestUI_HookNotification_SlowHook_ShowsRunningMessage
// Send a NotificationRunning event, advance time past threshold,
// assert status bar shows "Running SessionStart hook…".

// TestUI_HookNotification_FastHook_NoStatusMessage
// Send NotificationRunning then immediately NotificationComplete(proceed),
// assert status bar is empty (hookRunning guard prevented showing).

// TestUI_HookNotification_DenyShowsWarnMessage
// Send NotificationComplete(deny, reason="blocked"), assert status bar
// shows InfoTypeWarn with "SessionStart hook denied: blocked".

// TestUI_HookNotification_ErrorShowsWarnMessage
// Same pattern for Decision=="error".

// TestUI_HookNotification_ProceedClears_ExistingRunningMessage
// Set up a running message, send NotificationComplete(proceed),
// assert status bar is cleared.
```

## Acceptance Criteria

- A hook that takes > 300ms shows `Running <HookType> hook…` in the status bar
  while it executes; the message clears automatically when the hook completes.
- A hook that completes in < 300ms shows nothing.
- A deny result shows `<HookType> hook denied: <reason>` as a `InfoTypeWarn`
  status bar message with 10s TTL.
- A hook subprocess error shows `<HookType> hook error: <reason>` as
  `InfoTypeWarn` with 10s TTL.
- Async hooks produce no `NotificationRunning` event and no status bar message.
- A `NotificationComplete(proceed)` that arrives before the 300ms tick clears
  the `hookRunning` flag so the tick is a no-op.
- Existing hook tests are unaffected (nil publisher = no-op).
- `go build ./...` and `go test ./...` pass.

## Files to create or modify

| File | Change |
|---|---|
| `internal/hooks/notification.go` | New — `Notification`, `NotificationType` types |
| `internal/hooks/manager.go` | Add `publisher` field, `WithPublisher` option, publish calls |
| `internal/hooks/manager_test.go` | 4 new publisher tests |
| `internal/app/app.go` | Add `hookNotifications` broker, wire in `setupEvents`, pass to coordinator |
| `internal/agent/coordinator.go` | Accept broker, pass as `WithPublisher` to `hooks.NewManager` |
| `internal/ui/model/ui.go` | Handle `pubsub.Event[hooks.Notification]`, `hookRunning` flag, `hookRunningMsg`, `handleHookNotification` |
| `internal/ui/model/ui_test.go` | 5 new TUI feedback tests |

<!-- checkpoint: finalized -->
<!-- exit-state: Implementation complete. All files created/modified per spec, full test suite passes. Ready for finishing-branch. -->
<!-- key-files: internal/hooks/notification.go, internal/hooks/manager.go, internal/app/app.go, internal/agent/coordinator.go, internal/ui/model/ui.go, internal/hooks/manager_test.go -->
<!-- open-questions: none -->

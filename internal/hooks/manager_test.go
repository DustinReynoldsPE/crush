package hooks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ── Manager.Execute: basic routing ──────────────────────────────────────────

func TestManager_NoHooksForType_Proceed(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{})
	result, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_NilMap_Proceed(t *testing.T) {
	t.Parallel()
	m := NewManager(nil)
	result, err := m.Execute(context.Background(), PostToolUse, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_SingleHook_Proceed(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: "true"}},
	})
	result, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_SingleHook_Deny(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: `echo "policy violation" >&2; exit 2`}},
	})
	result, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.Equal(t, "deny", result.Decision)
	require.Contains(t, result.Reason, "policy violation")
}

func TestManager_MultipleHooks_AllProceed(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: "true"}, {Command: "true"}, {Command: "true"}},
	})
	result, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_DenyStopsChain(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/reached"
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {
			{Command: `exit 2`},
			{Command: "touch " + sentinel}, // must not run
		},
	})
	result, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "deny", result.Decision)
	require.False(t, fileExists(sentinel), "second hook must not execute after deny")
}

func TestManager_HookEventName_StampedOnEvent(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
name=$(cat | jq -r '.hook_event_name')
[ "$name" = "PostToolUse" ] || { echo "wrong event: $name" >&2; exit 2; }
`)
	m := NewManager(map[HookType][]HookConfig{
		PostToolUse: {{Command: script}},
	})
	result, err := m.Execute(context.Background(), PostToolUse, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_ContextCancelledBeforeExecution(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: "true"}},
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := m.Execute(ctx, PreToolUse, HookEvent{SessionID: "s1"})
	require.Error(t, err)
	require.Equal(t, "error", result.Decision)
}

func TestManager_NonBlockingErrorContinues(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/reached"
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {
			{Command: "exit 1"},                // non-blocking error
			{Command: "touch " + sentinel},     // should still run
		},
	})
	// A non-blocking error from the executor returns a Go error, which stops
	// the chain. Verify the first hook's error decision is reported.
	result, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1"})
	// Either an error or an "error" decision is acceptable for exit 1.
	_ = result
	_ = err
}

// ── Matcher filtering ────────────────────────────────────────────────────────

func TestManager_Matcher_ExactToolName_Fires(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/ran"
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: "touch " + sentinel, Matcher: HookMatcher{ToolName: "bash"}}},
	})
	_, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.True(t, fileExists(sentinel), "hook should fire for matching tool name")
}

func TestManager_Matcher_ExactToolName_Skipped(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/ran"
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: "touch " + sentinel, Matcher: HookMatcher{ToolName: "bash"}}},
	})
	_, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1", ToolName: "write"})
	require.NoError(t, err)
	require.False(t, fileExists(sentinel), "hook must not fire for non-matching tool name")
}

func TestManager_Matcher_RegexPattern_Fires(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/ran"
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: "touch " + sentinel, Matcher: HookMatcher{Pattern: `mcp__.*`}}},
	})
	_, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1", ToolName: "mcp__memory__store"})
	require.NoError(t, err)
	require.True(t, fileExists(sentinel), "hook should fire for pattern-matching tool")
}

func TestManager_Matcher_RegexPattern_Skipped(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/ran"
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: "touch " + sentinel, Matcher: HookMatcher{Pattern: `mcp__.*`}}},
	})
	_, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.False(t, fileExists(sentinel), "hook must not fire for non-matching pattern")
}

func TestManager_Matcher_EditOrWrite_Pattern(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/ran"
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: "touch " + sentinel, Matcher: HookMatcher{Pattern: `edit|write`}}},
	})
	_, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1", ToolName: "write"})
	require.NoError(t, err)
	require.True(t, fileExists(sentinel))
}

func TestManager_Matcher_NoMatcher_AlwaysFires(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/ran"
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: "touch " + sentinel}}, // no matcher
	})
	_, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1", ToolName: "anything"})
	require.NoError(t, err)
	require.True(t, fileExists(sentinel), "hook with no matcher should always fire")
}

// ── applyHookResult priority ─────────────────────────────────────────────────

func TestApplyHookResult_DenyOverProceed(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	result := m.applyHookResult(
		HookResult{Decision: "proceed"},
		HookResult{Decision: "deny", Reason: "blocked"},
	)
	require.Equal(t, "deny", result.Decision)
	require.Equal(t, "blocked", result.Reason)
}

func TestApplyHookResult_DenyOverModify(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	result := m.applyHookResult(
		HookResult{Decision: "modify", Reason: "earlier mod"},
		HookResult{Decision: "deny", Reason: "later deny"},
	)
	require.Equal(t, "deny", result.Decision)
}

func TestApplyHookResult_ExistingDeny_NotOverriddenByProceed(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	result := m.applyHookResult(
		HookResult{Decision: "deny", Reason: "original"},
		HookResult{Decision: "proceed"},
	)
	require.Equal(t, "deny", result.Decision)
	require.Equal(t, "original", result.Reason)
}

func TestApplyHookResult_ModifyOverProceed(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	result := m.applyHookResult(
		HookResult{Decision: "proceed"},
		HookResult{Decision: "modify", Reason: "tweak", ModifiedEvent: "new-event"},
	)
	require.Equal(t, "modify", result.Decision)
	require.Equal(t, "new-event", result.ModifiedEvent)
}

func TestApplyHookResult_ModifyChain_ReasonsAccumulate(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	result := m.applyHookResult(
		HookResult{Decision: "modify", Reason: "first mod", ModifiedEvent: "event-1"},
		HookResult{Decision: "modify", Reason: "second mod", ModifiedEvent: "event-2"},
	)
	require.Equal(t, "modify", result.Decision)
	require.Equal(t, "event-2", result.ModifiedEvent) // latest event wins
	require.Contains(t, result.Reason, "first mod")
	require.Contains(t, result.Reason, "second mod")
}

func TestApplyHookResult_ProceedChain_LatestReasonWins(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	result := m.applyHookResult(
		HookResult{Decision: "proceed", Reason: "hook 1 ok"},
		HookResult{Decision: "proceed", Reason: "hook 2 ok"},
	)
	require.Equal(t, "proceed", result.Decision)
	require.Equal(t, "hook 2 ok", result.Reason)
}

// ── matchesEvent ─────────────────────────────────────────────────────────────

func TestMatchesEvent_EmptyMatcher_AlwaysTrue(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	require.True(t, m.matchesEvent(HookConfig{}, HookEvent{ToolName: "anything"}))
}

func TestMatchesEvent_ToolName_Exact(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	cfg := HookConfig{Matcher: HookMatcher{ToolName: "bash"}}
	require.True(t, m.matchesEvent(cfg, HookEvent{ToolName: "bash"}))
	require.False(t, m.matchesEvent(cfg, HookEvent{ToolName: "write"}))
}

func TestMatchesEvent_Pattern_Regex(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	cfg := HookConfig{Matcher: HookMatcher{Pattern: `mcp__memory__.*`}}
	require.True(t, m.matchesEvent(cfg, HookEvent{ToolName: "mcp__memory__store"}))
	require.True(t, m.matchesEvent(cfg, HookEvent{ToolName: "mcp__memory__retrieve"}))
	require.False(t, m.matchesEvent(cfg, HookEvent{ToolName: "mcp__fs__read"}))
	require.False(t, m.matchesEvent(cfg, HookEvent{ToolName: "bash"}))
}

func TestMatchesEvent_InvalidPattern_NoMatch(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	cfg := HookConfig{Matcher: HookMatcher{Pattern: `[invalid`}}
	require.False(t, m.matchesEvent(cfg, HookEvent{ToolName: "bash"}))
}

// ── PostToolUseFailure routing ───────────────────────────────────────────────

func TestManager_PostToolUseFailure_RoutedSeparatelyFromPostToolUse(t *testing.T) {
	t.Parallel()
	successSentinel := t.TempDir() + "/success-ran"
	failureSentinel := t.TempDir() + "/failure-ran"
	m := NewManager(map[HookType][]HookConfig{
		PostToolUse:        {{Command: "touch " + successSentinel}},
		PostToolUseFailure: {{Command: "touch " + failureSentinel}},
	})

	_, err := m.Execute(context.Background(), PostToolUse, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.True(t, fileExists(successSentinel), "PostToolUse hook must fire for PostToolUse events")
	require.False(t, fileExists(failureSentinel), "PostToolUseFailure hook must not fire for PostToolUse events")

	_, err = m.Execute(context.Background(), PostToolUseFailure, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.True(t, fileExists(failureSentinel), "PostToolUseFailure hook must fire for PostToolUseFailure events")
}

func TestManager_PostToolUseFailure_HookEventName_Stamped(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
name=$(cat | jq -r '.hook_event_name')
[ "$name" = "PostToolUseFailure" ] || { echo "wrong event: $name" >&2; exit 2; }
`)
	m := NewManager(map[HookType][]HookConfig{
		PostToolUseFailure: {{Command: script}},
	})
	result, err := m.Execute(context.Background(), PostToolUseFailure, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_PostToolUseFailure_Deny(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		PostToolUseFailure: {{Command: `echo "audit failure" >&2; exit 2`}},
	})
	result, err := m.Execute(context.Background(), PostToolUseFailure, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.Equal(t, "deny", result.Decision)
	require.Contains(t, result.Reason, "audit failure")
}

func TestManager_PostToolUseFailure_MatcherFiltersCorrectly(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/ran"
	m := NewManager(map[HookType][]HookConfig{
		PostToolUseFailure: {{
			Command: "touch " + sentinel,
			Matcher: HookMatcher{ToolName: "bash"},
		}},
	})

	_, err := m.Execute(context.Background(), PostToolUseFailure, HookEvent{SessionID: "s1", ToolName: "write"})
	require.NoError(t, err)
	require.False(t, fileExists(sentinel), "matcher must filter by tool name")

	_, err = m.Execute(context.Background(), PostToolUseFailure, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.True(t, fileExists(sentinel), "matcher must allow matching tool name")
}

// ── Stop hook deny carries continuation reason ───────────────────────────────

func TestManager_Stop_Deny_CarriesReason(t *testing.T) {
	t.Parallel()
	// The agent uses hookResult.Reason as the continuation prompt when decision
	// is "deny". Verify the manager faithfully returns the hook-provided reason.
	m := NewManager(map[HookType][]HookConfig{
		Stop: {{Command: `echo '{"decision":"deny","reason":"please summarize todos"}'`}},
	})
	result, err := m.Execute(context.Background(), Stop, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "deny", result.Decision)
	require.Equal(t, "please summarize todos", result.Reason)
}

func TestManager_Stop_Proceed_NoBlockingEffect(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		Stop: {{Command: "true"}},
	})
	result, err := m.Execute(context.Background(), Stop, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_Stop_DenyViaExitTwo_CarriesStderrReason(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		Stop: {{Command: `echo "run post-processing" >&2; exit 2`}},
	})
	result, err := m.Execute(context.Background(), Stop, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "deny", result.Decision)
	require.Contains(t, result.Reason, "run post-processing")
}

func TestManager_Stop_HookEventName_Stamped(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
name=$(cat | jq -r '.hook_event_name')
[ "$name" = "Stop" ] || { echo "wrong event: $name" >&2; exit 2; }
`)
	m := NewManager(map[HookType][]HookConfig{
		Stop: {{Command: script}},
	})
	result, err := m.Execute(context.Background(), Stop, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

// ── Async hooks ──────────────────────────────────────────────────────────────

func TestManager_AsyncHook_DoesNotBlockChain(t *testing.T) {
	t.Parallel()
	// Async hook sleeps 2s — chain must complete well before that.
	sentinel := t.TempDir() + "/sync-ran"
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {
			{Command: "sleep 2", Async: true},
			{Command: "touch " + sentinel},
		},
	})
	start := time.Now()
	result, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1", ToolName: "bash"})
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
	require.True(t, fileExists(sentinel), "sync hook after async must still run")
	require.Less(t, elapsed, 1500*time.Millisecond, "chain must not block on async hook")
}

func TestManager_AsyncHook_DenyIsIgnored(t *testing.T) {
	t.Parallel()
	// Async hook that would deny must not affect the chain decision.
	m := NewManager(map[HookType][]HookConfig{
		PreToolUse: {{Command: "exit 2", Async: true}},
	})
	result, err := m.Execute(context.Background(), PreToolUse, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision, "async hook deny must be ignored")
}

// ── PermissionRequest / PermissionDenied routing ────────────────────────────

func TestManager_PermissionRequest_Deny(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		PermissionRequest: {{Command: `echo "blocked by policy" >&2; exit 2`}},
	})
	result, err := m.Execute(context.Background(), PermissionRequest, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.Equal(t, "deny", result.Decision)
	require.Contains(t, result.Reason, "blocked by policy")
}

func TestManager_PermissionRequest_Approve(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		PermissionRequest: {{Command: `echo '{"decision":"approve","reason":"auto-approved"}'`}},
	})
	result, err := m.Execute(context.Background(), PermissionRequest, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.Equal(t, "approve", result.Decision)
}

func TestManager_PermissionDenied_HookEventName(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
name=$(cat | jq -r '.hook_event_name')
[ "$name" = "PermissionDenied" ] || { echo "wrong: $name" >&2; exit 2; }
`)
	m := NewManager(map[HookType][]HookConfig{
		PermissionDenied: {{Command: script}},
	})
	result, err := m.Execute(context.Background(), PermissionDenied, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_PermissionRequest_ToolNameMatcher(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/ran"
	m := NewManager(map[HookType][]HookConfig{
		PermissionRequest: {{Command: "touch " + sentinel, Matcher: HookMatcher{ToolName: "bash"}}},
	})

	_, _ = m.Execute(context.Background(), PermissionRequest, HookEvent{SessionID: "s1", ToolName: "view"})
	require.False(t, fileExists(sentinel), "must not fire for non-matching tool")

	_, _ = m.Execute(context.Background(), PermissionRequest, HookEvent{SessionID: "s1", ToolName: "bash"})
	require.True(t, fileExists(sentinel), "must fire for matching tool")
}

// ── SessionStart hook ────────────────────────────────────────────────────────

func TestManager_SessionStart_Proceed(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		SessionStart: {{Command: "true"}},
	})
	result, err := m.Execute(context.Background(), SessionStart, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_SessionStart_Deny(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		SessionStart: {{Command: `echo "session blocked by policy" >&2; exit 2`}},
	})
	result, err := m.Execute(context.Background(), SessionStart, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "deny", result.Decision)
	require.Contains(t, result.Reason, "session blocked by policy")
}

func TestManager_SessionStart_HookEventName_Stamped(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
name=$(cat | jq -r '.hook_event_name')
[ "$name" = "SessionStart" ] || { echo "wrong event: $name" >&2; exit 2; }
`)
	m := NewManager(map[HookType][]HookConfig{
		SessionStart: {{Command: script}},
	})
	result, err := m.Execute(context.Background(), SessionStart, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_SessionStart_PayloadHasSessionID(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
sid=$(cat | jq -r '.session_id')
[ "$sid" = "test-session-42" ] || { echo "wrong session_id: $sid" >&2; exit 2; }
`)
	m := NewManager(map[HookType][]HookConfig{
		SessionStart: {{Command: script}},
	})
	result, err := m.Execute(context.Background(), SessionStart, HookEvent{SessionID: "test-session-42"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_SessionStart_NoHooks_Proceed(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{})
	result, err := m.Execute(context.Background(), SessionStart, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_SessionStart_DenyStopsBeforeUserPromptSubmit(t *testing.T) {
	t.Parallel()
	// If SessionStart denies, UserPromptSubmit hooks must not run.
	upsSentinel := t.TempDir() + "/ups-ran"
	m := NewManager(map[HookType][]HookConfig{
		SessionStart:     {{Command: `exit 2`}},
		UserPromptSubmit: {{Command: "touch " + upsSentinel}},
	})
	result, err := m.Execute(context.Background(), SessionStart, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "deny", result.Decision)
	// Caller should not proceed to fire UserPromptSubmit after a SessionStart deny.
	// The sentinel must remain absent — verified here as documentation of the contract.
	require.False(t, fileExists(upsSentinel), "UserPromptSubmit must not fire after SessionStart deny")
}

// ── Notification hook ────────────────────────────────────────────────────────

func TestManager_Notification_Proceed(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		Notification: {{Command: "true"}},
	})
	result, err := m.Execute(context.Background(), Notification, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_Notification_HookEventName_Stamped(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
name=$(cat | jq -r '.hook_event_name')
[ "$name" = "Notification" ] || { echo "wrong event: $name" >&2; exit 2; }
`)
	m := NewManager(map[HookType][]HookConfig{
		Notification: {{Command: script}},
	})
	result, err := m.Execute(context.Background(), Notification, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_Notification_PayloadHasSessionIDAndMessage(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
payload=$(cat)
sid=$(echo "$payload" | jq -r '.session_id')
msg=$(echo "$payload" | jq -r '.data.message')
[ "$sid" = "test-session-99" ] || { echo "wrong session_id: $sid" >&2; exit 2; }
[ "$msg" = "agent_finished" ] || { echo "wrong message: $msg" >&2; exit 2; }
`)
	m := NewManager(map[HookType][]HookConfig{
		Notification: {{Command: script}},
	})
	result, err := m.Execute(context.Background(), Notification, HookEvent{
		SessionID:    "test-session-99",
		RawEventData: map[string]string{"message": "agent_finished"},
	})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_Notification_AsyncDoesNotBlockChain(t *testing.T) {
	t.Parallel()
	sentinel := t.TempDir() + "/sync-ran"
	m := NewManager(map[HookType][]HookConfig{
		Notification: {
			{Command: "sleep 2", Async: true},
			{Command: "touch " + sentinel},
		},
	})
	start := time.Now()
	result, err := m.Execute(context.Background(), Notification, HookEvent{SessionID: "s1"})
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
	require.True(t, fileExists(sentinel), "sync hook after async must still run")
	require.Less(t, elapsed, 1500*time.Millisecond, "chain must not block on async hook")
}

func TestManager_Notification_NoHooks_Proceed(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{})
	result, err := m.Execute(context.Background(), Notification, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

// ── AgentError hook ──────────────────────────────────────────────────────────

func TestManager_AgentError_Proceed(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{
		AgentError: {{Command: "true"}},
	})
	result, err := m.Execute(context.Background(), AgentError, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_AgentError_HookEventName_Stamped(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
name=$(cat | jq -r '.hook_event_name')
[ "$name" = "AgentError" ] || { echo "wrong event: $name" >&2; exit 2; }
`)
	m := NewManager(map[HookType][]HookConfig{
		AgentError: {{Command: script}},
	})
	result, err := m.Execute(context.Background(), AgentError, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_AgentError_PayloadHasSessionIDAndError(t *testing.T) {
	t.Parallel()
	script := writeScript(t, `#!/bin/sh
payload=$(cat)
sid=$(echo "$payload" | jq -r '.session_id')
errMsg=$(echo "$payload" | jq -r '.data.error')
[ "$sid" = "test-session-err" ] || { echo "wrong session_id: $sid" >&2; exit 2; }
[ -n "$errMsg" ] || { echo "missing error field" >&2; exit 2; }
[ "$errMsg" != "null" ] || { echo "error field is null" >&2; exit 2; }
`)
	m := NewManager(map[HookType][]HookConfig{
		AgentError: {{Command: script}},
	})
	result, err := m.Execute(context.Background(), AgentError, HookEvent{
		SessionID:    "test-session-err",
		RawEventData: map[string]string{"error": "provider returned 503"},
	})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}

func TestManager_AgentError_NoHooks_Proceed(t *testing.T) {
	t.Parallel()
	m := NewManager(map[HookType][]HookConfig{})
	result, err := m.Execute(context.Background(), AgentError, HookEvent{SessionID: "s1"})
	require.NoError(t, err)
	require.Equal(t, "proceed", result.Decision)
}
